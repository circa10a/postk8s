package controller

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	mailform "github.com/circa10a/go-mailform"
	mailformv1alpha1 "github.com/circa10a/postk8s/api/v1alpha1"
)

// MailReconciler reconciles a Mail object
type MailReconciler struct {
	client.Client
	MailformClient *mailform.Client
	Scheme         *runtime.Scheme
	SyncInterval   time.Duration
}

// Definitions to manage status conditions
const (
	// typeValidationMail represents the status of the Mail validation
	typeValidationMail = "Validation"
	// typeValidationMail represents the status of the Mail fuilfillment
	typeFulfillmentMail = "Fuilfillment"
)

// +kubebuilder:rbac:groups=mailform.circa10a.github.io,resources=mails,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mailform.circa10a.github.io,resources=mails/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mailform.circa10a.github.io,resources=mails/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MailReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Get the mail object
	mail, err := r.loadMail(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Nothing to do if mail is already sent or cancelled
	if mail.Status.Sent || mail.Status.State == mailform.StatusCancelled {
		log.Info("order sent/cancelled", "orderID", mail.Status.ID)
		return ctrl.Result{}, nil
	}

	// Create order if it doesn't exist
	if mail.Status.ID == "" {
		orderID, err := r.createOrder(ctx, mail)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created mail order", "orderID", orderID)
	}

	// Get order details
	order, err := r.fetchOrder(mail.Status.ID)
	if err != nil {
		log.Error(err, "error fetching order", "orderID", mail.Status.ID)
		return ctrl.Result{}, err
	}

	// Update status fields if there are any order updates
	err = r.updateStatusFromOrder(ctx, mail, order)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("got status from order, requeuing",
		"orderID", mail.Status.ID,
		"state", mail.Status.State,
		"sent", mail.Status.Sent,
		"requeueAfter", r.SyncInterval,
	)

	return ctrl.Result{RequeueAfter: r.SyncInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MailReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mailformv1alpha1.Mail{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("mail").
		Complete(r)
}

// loadMail fetches the Mail object.
func (r *MailReconciler) loadMail(ctx context.Context, key types.NamespacedName) (*mailformv1alpha1.Mail, error) {
	mail := &mailformv1alpha1.Mail{}

	err := r.Get(ctx, key, mail)
	if err != nil {
		return nil, err
	}

	return mail, nil
}

// buildOrderInput builds the external API order input from the Mail spec.
func (r *MailReconciler) buildOrderInput(mail *mailformv1alpha1.Mail) mailform.OrderInput {
	return mailform.OrderInput{
		FilePath:          mail.Spec.FilePath,
		URL:               mail.Spec.URL,
		CustomerReference: mail.Spec.CustomerReference,
		Service:           mail.Spec.Service,
		Webhook:           mail.Spec.Webhook,
		Company:           mail.Spec.Company,
		Simplex:           mail.Spec.Simplex,
		Color:             mail.Spec.Color,
		Flat:              mail.Spec.Flat,
		Message:           mail.Spec.Message,
		ToName:            mail.Spec.To.Name,
		ToOrganization:    mail.Spec.To.Organization,
		ToAddress1:        mail.Spec.To.Address1,
		ToAddress2:        mail.Spec.To.Address2,
		ToCity:            mail.Spec.To.City,
		ToState:           mail.Spec.To.State,
		ToPostcode:        mail.Spec.To.Postcode,
		ToCountry:         mail.Spec.To.Country,
		FromName:          mail.Spec.From.Name,
		FromOrganization:  mail.Spec.From.Organization,
		FromAddress1:      mail.Spec.From.Address1,
		FromAddress2:      mail.Spec.From.Address2,
		FromCity:          mail.Spec.From.City,
		FromState:         mail.Spec.From.State,
		FromPostcode:      mail.Spec.From.Postcode,
		FromCountry:       mail.Spec.From.Country,
	}
}

// createOrder with create an order.
func (r *MailReconciler) createOrder(ctx context.Context, mail *mailformv1alpha1.Mail) (string, error) {
	orderInput := r.buildOrderInput(mail)
	err := orderInput.Validate()
	if err != nil {
		// persist validation failure in status and stop reconciling
		meta.SetStatusCondition(&mail.Status.Conditions, metav1.Condition{
			Type:               typeValidationMail,
			Status:             metav1.ConditionFalse,
			Reason:             "ValidationFailed",
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})

		err2 := r.Status().Update(ctx, mail)
		if err2 != nil {
			return "", err
		}

		return "", err2
	}

	order, err := r.MailformClient.CreateOrder(orderInput)
	if err != nil {
		return "", err
	}

	mail.Status.ID = order.Data.ID

	err = r.Status().Update(ctx, mail)
	if err != nil {
		return order.Data.ID, err
	}

	return order.Data.ID, nil
}

// fetchOrder retrieves the order from the external API.
func (r *MailReconciler) fetchOrder(orderID string) (*mailform.Order, error) {
	return r.MailformClient.GetOrder(orderID)
}

// updateStatusFromOrder maps the external order into Mail.Status and persists it.
func (r *MailReconciler) updateStatusFromOrder(ctx context.Context, mail *mailformv1alpha1.Mail, order *mailform.Order) error {
	mail.Status.State = order.Data.State
	mail.Status.Sent = order.Data.State == mailform.StatusFulfilled
	mail.Status.Total = order.Data.Total
	mail.Status.Created = metav1.NewTime(order.Data.Created)
	mail.Status.Modified = metav1.NewTime(order.Data.Modified)
	mail.Status.Cancelled = metav1.NewTime(order.Data.Cancelled)
	mail.Status.CancellationReason = order.Data.CancellationReason

	now := metav1.Now()

	meta.SetStatusCondition(&mail.Status.Conditions, metav1.Condition{
		Type:               typeValidationMail,
		Status:             metav1.ConditionTrue,
		Reason:             "ValidationPassed",
		Message:            "Mail spec validated successfully",
		LastTransitionTime: now,
	})

	if mail.Status.Sent {
		meta.SetStatusCondition(&mail.Status.Conditions, metav1.Condition{
			Type:               typeFulfillmentMail,
			Status:             metav1.ConditionTrue,
			Reason:             "Fulfilled",
			Message:            "Mail order fulfilled",
			LastTransitionTime: now,
		})
	} else {
		meta.SetStatusCondition(&mail.Status.Conditions, metav1.Condition{
			Type:               typeFulfillmentMail,
			Status:             metav1.ConditionFalse,
			Reason:             "AwaitingFulfillment",
			Message:            "Mail order not fulfilled yet",
			LastTransitionTime: now,
		})
	}

	err := r.Status().Update(ctx, mail)
	if err != nil {
		return err
	}

	return nil
}
