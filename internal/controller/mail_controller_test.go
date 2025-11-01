package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/circa10a/go-mailform"
	mailformv1alpha1 "github.com/circa10a/postk8s/api/v1alpha1"
)

var _ = Describe("Mail Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		mail := &mailformv1alpha1.Mail{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Mail")
			err := k8sClient.Get(ctx, typeNamespacedName, mail)
			if err != nil && errors.IsNotFound(err) {
				resource := &mailformv1alpha1.Mail{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &mailformv1alpha1.Mail{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Mail")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")

			mailformClient, err := mailform.New(&mailform.Config{})
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mailformClient,
				SyncInterval:   time.Second * 5,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
