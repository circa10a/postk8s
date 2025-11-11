package controller

import (
	"context"
	"time"

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

			key := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

			resource := &mailformv1alpha1.Mail{}
			err := k8sClient.Get(ctx, key, resource)
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

		It("should fail if spec is invalid", func() {
			By("Reconciling a resource with an invalid mail spec")

			key := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

			resource := &mailformv1alpha1.Mail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: mailformv1alpha1.MailSpec{
					Service: "RESPECT_MUH_AUTHORITAH",
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

			fetched := &mailformv1alpha1.Mail{}
			Expect(k8sClient.Get(ctx, key, fetched)).To(Succeed())

			controllerReconciler := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mockMailformClient{},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: key,
			})
			Expect(err).To(HaveOccurred())
			Expect(fetched.Status.Valid).To(BeFalse())
		})

		It("should not do anything if already sent", func() {
			By("Reconciling a resource with sent in the status")

			key := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

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
			Expect(k8sClient.Get(ctx, key, fetched)).To(Succeed())
			Expect(fetched.Status.Sent).To(BeTrue())

			controllerReconciler := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mockMailformClient{},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: key,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create an order and update status if valid and unsent", func() {
			ctx := context.Background()
			key := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

			order := &mailform.Order{
				Success: true,
			}
			order.Data.ID = "order-123"
			order.Data.State = mailform.StatusAwaitingFulfillment
			order.Data.Created = time.Now()
			order.Data.Modified = time.Now()
			order.Data.Total = 10

			resource := &mailformv1alpha1.Mail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: mailformv1alpha1.MailSpec{
					Service: "USPS_PRIORITY",
					URL:     "https://pdfobject.com/pdf/sample.pdf",
					To: &mailformv1alpha1.Address{
						Name:     "to-name",
						Address1: "123 Main St",
						City:     "City",
						Country:  "US",
						Postcode: "11111",
						State:    "CA",
					},
					From: &mailformv1alpha1.Address{
						Name:     "from-name",
						Address1: "321 Other St",
						City:     "City",
						Country:  "US",
						Postcode: "22222",
						State:    "CA",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			controller := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mockMailformClient{output: order},
				SyncInterval:   1 * time.Second,
			}

			result, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(1 * time.Second))

			fetched := &mailformv1alpha1.Mail{}
			Expect(k8sClient.Get(ctx, key, fetched)).To(Succeed())
			Expect(fetched.Status.ID).To(Equal("order-123"))
			Expect(fetched.Status.Valid).To(BeTrue())
			Expect(fetched.Status.Sent).To(BeFalse())
		})

		It("should update sent status when external order is fulfilled", func() {
			ctx := context.Background()
			key := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

			order := &mailform.Order{Success: true}
			order.Data.ID = "order-fulfilled"
			order.Data.State = mailform.StatusFulfilled
			order.Data.Created = time.Now()
			order.Data.Modified = time.Now()
			order.Data.Total = 20

			resource := &mailformv1alpha1.Mail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: mailformv1alpha1.MailSpec{
					Service: "USPS_PRIORITY",
					URL:     "https://pdfobject.com/pdf/sample.pdf",
					To: &mailformv1alpha1.Address{
						Name:     "to",
						Address1: "a",
						City:     "b",
						Country:  "US",
						Postcode: "12345",
						State:    "CA",
					},
					From: &mailformv1alpha1.Address{
						Name:     "from",
						Address1: "a",
						City:     "b",
						Country:  "US",
						Postcode: "54321",
						State:    "CA",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			resource.Status.ID = "order-fulfilled"
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			controller := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mockMailformClient{output: order},
				SyncInterval:   2 * time.Second,
			}

			result, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(2 * time.Second))

			fetched := &mailformv1alpha1.Mail{}
			Expect(k8sClient.Get(ctx, key, fetched)).To(Succeed())
			Expect(fetched.Status.Sent).To(BeTrue())
			Expect(fetched.Status.State).To(Equal(mailform.StatusFulfilled))
		})

		It("should return an error if mailform API fails", func() {
			ctx := context.Background()
			key := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

			resource := &mailformv1alpha1.Mail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: mailformv1alpha1.MailSpec{
					Service: "USPS_PRIORITY",
					URL:     "https://pdfobject.com/pdf/sample.pdf",
					To: &mailformv1alpha1.Address{
						Name:     "to",
						Address1: "a",
						City:     "b",
						Country:  "US",
						Postcode: "12345",
						State:    "CA",
					},
					From: &mailformv1alpha1.Address{
						Name:     "from",
						Address1: "a",
						City:     "b",
						Country:  "US",
						Postcode: "54321",
						State:    "CA",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			mailformClient := mockMailformClient{mockErr: errors.NewBadRequest("mailform error")}
			controller := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mailformClient,
				SyncInterval:   1 * time.Second,
			}

			_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mailform error"))
		})

		It("should reconcile multiple times with a short sync interval", func() {
			ctx := context.Background()
			key := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

			// Mock order that starts in a pending state
			order := &mailform.Order{
				Success: true,
			}
			order.Data.ID = "order-requeue"
			order.Data.State = mailform.StatusAwaitingFulfillment
			order.Data.Created = time.Now()
			order.Data.Modified = time.Now()
			order.Data.Total = 10

			resource := &mailformv1alpha1.Mail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: mailformv1alpha1.MailSpec{
					Service: "USPS_PRIORITY",
					URL:     "https://pdfobject.com/pdf/sample.pdf",
					To: &mailformv1alpha1.Address{
						Name:     "to",
						Address1: "a",
						City:     "b",
						Country:  "US",
						Postcode: "12345",
						State:    "CA",
					},
					From: &mailformv1alpha1.Address{
						Name:     "from",
						Address1: "a",
						City:     "b",
						Country:  "US",
						Postcode: "54321",
						State:    "CA",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			mockClient := &mockMailformClient{output: order}

			controller := &MailReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				MailformClient: mockClient,
				SyncInterval:   200 * time.Millisecond, // short interval for testing
			}

			// First reconcile: should create the order and requeue
			firstResult, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())
			Expect(firstResult.RequeueAfter).To(Equal(200 * time.Millisecond))

			fetched := &mailformv1alpha1.Mail{}
			Expect(k8sClient.Get(ctx, key, fetched)).To(Succeed())
			Expect(fetched.Status.ID).To(Equal("order-requeue"))
			Expect(fetched.Status.Sent).To(BeFalse())

			// Simulate order fulfillment before the next reconcile
			order.Data.State = mailform.StatusFulfilled
			order.Data.Modified = time.Now().Add(100 * time.Millisecond)

			// Wait for requeue interval
			time.Sleep(250 * time.Millisecond)

			// Second reconcile: should mark the mail as Sent
			secondResult, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())
			Expect(secondResult.RequeueAfter).To(Equal(200 * time.Millisecond))

			Expect(k8sClient.Get(ctx, key, fetched)).To(Succeed()) // did it work
			Expect(fetched.Status.Sent).To(BeTrue())
			Expect(fetched.Status.State).To(Equal(mailform.StatusFulfilled))
		})
	})
})
