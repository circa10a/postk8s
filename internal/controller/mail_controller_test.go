package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/circa10a/go-mailform"
	mailformv1alpha1 "github.com/circa10a/postk8s/api/v1alpha1"
)

// mockMailformClient is for mocking mailform responses
type mockMailformClient struct {
	output  *mailform.Order
	mockErr error
}

// CreateOrder is for creating mock orders. Will return mockErr if not nill
func (m mockMailformClient) CreateOrder(o mailform.OrderInput) (*mailform.Order, error) {
	if m.mockErr != nil {
		return m.output, m.mockErr
	}

	return m.output, nil
}

// GetOrder is for fetching mock orders. Will return mockErr if not nill
func (m mockMailformClient) GetOrder(o string) (*mailform.Order, error) {
	if m.mockErr != nil {
		return m.output, m.mockErr
	}

	return m.output, nil
}

const (
	resourceName  = "test-resource"
	namespaceName = "default"
)

var _ = Describe("Mail Controller", func() {
	Context("When reconciling a resource", func() {

		BeforeEach(func() {})

		AfterEach(func() {
			ctx := context.Background()

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespaceName,
			}

			resource := &mailformv1alpha1.Mail{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil {
				if errors.IsNotFound(err) {
					// nothing to clean up
					return
				}
				// unexpected error
				Expect(err).NotTo(HaveOccurred())
			}

			By("Cleanup the specific resource instance Mail")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should fail if no to/from addresses are provided", func() {
			By("Reconciling a resource with sent in the status")

			resource := &mailformv1alpha1.Mail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(MatchError(ContainSubstring("spec.from: Required value")))
			Expect(k8sClient.Create(ctx, resource)).To(MatchError(ContainSubstring("spec.to: Required value")))

		})

		It("should not do anything if already sent", func() {
			By("Reconciling a resource with sent in the status")

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespaceName,
			}

			resource := &mailformv1alpha1.Mail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: mailformv1alpha1.MailSpec{
					Service: "USPS_PRIORITY",
					URL:     "https://pdfobject.com/pdf/sample.pdf",
					To: &mailformv1alpha1.Address{
						Name:     "test",
						Address1: "test",
						City:     "test",
						Country:  "test",
						Postcode: "test",
						State:    "test",
					},
					From: &mailformv1alpha1.Address{
						Name:     "test",
						Address1: "test",
						City:     "test",
						Country:  "test",
						Postcode: "test",
						State:    "test",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			// Manually set status.sent = true and update via the status client
			resource.Status.Sent = true
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			// Now confirm it was updated
			fetched := &mailformv1alpha1.Mail{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, fetched)).To(Succeed())
			Expect(fetched.Status.Sent).To(BeTrue())

			controllerReconciler := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mockMailformClient{},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
