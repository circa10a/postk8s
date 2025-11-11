package controller

import (
	"context"
	"strconv"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	mailform "github.com/circa10a/go-mailform"
	mailformv1alpha1 "github.com/circa10a/postk8s/api/v1alpha1"
)

// MailformIface is an interface to Create/Get orders from Mailform.
type MailformIface interface {
	CreateOrder(o mailform.OrderInput) (*mailform.Order, error)
	GetOrder(o string) (*mailform.Order, error)
	CancelOrder(o string) error
}

// MailReconciler reconciles a Mail object
type MailReconciler struct {
	client.Client
	MailformClient MailformIface
	Scheme         *runtime.Scheme
	SyncInterval   time.Duration
}

// Definitions to manage status conditions
const (
	// typeValidationMail represents the status of the Mail fuilfillment
	typeFulfillmentMail = "Fulfillment"
	// Finalizer for ensuring safe to delete by validated mail was sent/cancelled
	mailSentOrCancelledFinalizerName = "mailform.circa10a.github.io/mail-sent-or-cancelled-finalizer"
	// This is our exception annotation to override the finalizer so mail can be deleted without talking to mailform.
	skipCancellationOnDeleteAnnotation = "mailform.circa10a.github.io/skip-cancellation-on-delete"
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

	// Add finalizer for ensuring mail was sent or cancelled.
	err = r.ensureMailSentOrCancelledFinalizer(ctx, mail)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Ensure finalizers are met
	done, err := r.handleDeletion(ctx, mail)
	if err != nil {
		return ctrl.Result{}, err
	}

	if done {
		return ctrl.Result{}, nil
	}

	// Nothing to do if mail is already sent or cancelled
	if mail.Status.Sent || mail.Status.State == mailform.StatusCancelled {
		log.Info("order sent/cancelled", "name", req.Name, "orderID", mail.Status.ID)
		return ctrl.Result{}, nil
	}

	orderInput := r.buildOrderInput(mail)
	err = orderInput.Validate()

	if err != nil {
		log.Error(err, "mail spec invalid, skipping reconciliation", "name", req.Name)
		return ctrl.Result{}, err
	}

	// Mailspec is valid let's ensure it's updated only once
	if !mail.Status.Valid {
		mail.Status.Valid = true
		err = r.Status().Update(ctx, mail)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create order if it doesn't exist
	if mail.Status.ID == "" {
		orderID, err := r.createOrder(ctx, mail, &orderInput)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created mail order", "name", req.Name, "orderID", orderID)
	}

	// Get order details
	order, err := r.fetchOrder(mail.Status.ID)
	if err != nil {
		log.Error(err, "error fetching order", "name", req.Name, "orderID", mail.Status.ID)
		return ctrl.Result{}, err
	}

	// Update status fields if there are any order updates
	err = r.updateStatusFromOrder(ctx, mail, order)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("got status from order, requeuing",
		"name", req.Name,
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

// ensureMailSentOrCancelledFinalizer adds the finalizer responsible for not allowing delete until sent/cancelled.
func (r *MailReconciler) ensureMailSentOrCancelledFinalizer(ctx context.Context, mail *mailformv1alpha1.Mail) error {
	if mail.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(mail, mailSentOrCancelledFinalizerName) {
		controllerutil.AddFinalizer(mail, mailSentOrCancelledFinalizerName)
		return r.Update(ctx, mail)
	}

	return nil
}

// Ensure finalizer conditions are met and can be deleted
func (r *MailReconciler) handleDeletion(ctx context.Context, mail *mailformv1alpha1.Mail) (bool, error) {
	log := logf.FromContext(ctx)

	if mail.DeletionTimestamp.IsZero() {
		return false, nil
	}

	if controllerutil.ContainsFinalizer(mail, mailSentOrCancelledFinalizerName) {
		// Check for skip override
		val := mail.Annotations[skipCancellationOnDeleteAnnotation]
		skip, _ := strconv.ParseBool(val)
		if skip {
			log.Info("skip-cancellation annotation present, skipping cancellation for", "orderID", mail.Status.ID, "name", mail.Name)
			controllerutil.RemoveFinalizer(mail, mailSentOrCancelledFinalizerName)
			return true, r.Update(ctx, mail)
		}

		// Fetch latest state from Mailform if there is an order ID
		order := &mailform.Order{}
		var err error
		if mail.Status.ID != "" {
			order, err = r.fetchOrder(mail.Status.ID)
			if err != nil {
				log.Error(err, "failed to fetch order for deletion check", "orderID", mail.Status.ID, "name", mail.Name)
				return true, err
			}
		}

		// Only cancel if not already sent/cancelled
		if order == nil || (order.Data.State != mailform.StatusFulfilled && order.Data.State != mailform.StatusCancelled) {
			if mail.Status.ID != "" {
				err := r.cancelOrder(mail.Status.ID)
				if err != nil {
					log.Error(err, "failed to cancel order", "orderID", mail.Status.ID, "name", mail.Name)
					return true, err
				}
				log.Info("order cancelled", "orderID", mail.Status.ID, "name", mail.Name)
			}
		}

		// Remove finalizer after cancelling or if already sent
		controllerutil.RemoveFinalizer(mail, mailSentOrCancelledFinalizerName)
		if err := r.Update(ctx, mail); err != nil {
			return true, err
		}
	}

	return true, nil
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
func (r *MailReconciler) createOrder(ctx context.Context, mail *mailformv1alpha1.Mail, orderInput *mailform.OrderInput) (string, error) {
	order, err := r.MailformClient.CreateOrder(*orderInput)
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

// fetchOrder retrieves the order via the external API.
func (r *MailReconciler) fetchOrder(orderID string) (*mailform.Order, error) {
	return r.MailformClient.GetOrder(orderID)
}

// cancelOrder cancels the order via the external API.
func (r *MailReconciler) cancelOrder(orderID string) error {
	return r.MailformClient.CancelOrder(orderID)
}

// updateStatusFromOrder maps the external order into Mail.Status and persists it.
func (r *MailReconciler) updateStatusFromOrder(ctx context.Context, mail *mailformv1alpha1.Mail, order *mailform.Order) error {
	mail.Status.Sent = order.Data.State == mailform.StatusFulfilled
	mail.Status.State = order.Data.State
	mail.Status.Total = order.Data.Total
	mail.Status.Created = metav1.NewTime(order.Data.Created)
	mail.Status.Modified = metav1.NewTime(order.Data.Modified)
	mail.Status.Cancelled = metav1.NewTime(order.Data.Cancelled)
	mail.Status.CancellationReason = order.Data.CancellationReason

	now := metav1.Now()

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
