package v1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// These tests are written in BDD-style using Ginkgo framework. Refer to
// http://onsi.github.io/ginkgo to learn more.
var k8sClient client.Client
var _ = Describe("Job", func() {
	var (
		key              types.NamespacedName
		created, fetched *Job
	)

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	// Add Tests for OpenAPI validation (or additional CRD features) specified in
	// your API definition.
	// Avoid adding tests for vanilla CRUD operations because they would
	// test Kubernetes API server, which isn't the goal here.
	Context("Create API", func() {

		It("should create an object successfully", func() {

			key = types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			created = &Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
			}

			By("creating an API obj")
			Expect(k8sClient.Create(context.Background(), created)).To(Succeed())

			fetched = &Job{}
			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched).To(Equal(created))

			By("deleting the created object")
			Expect(k8sClient.Delete(context.Background(), created)).To(Succeed())
			Expect(k8sClient.Get(context.Background(), key, created)).ToNot(Succeed())
		})

		It("should correctly handle logs finalizers", func() {
			op := &Job{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
				},
			}
			Expect(IsOperationBeingDeleted(op)).To(BeTrue())

			controllerutil.AddFinalizer(op, OperationLogsFinalizer)
			Expect(len(op.GetFinalizers())).To(Equal(1))
			Expect(controllerutil.ContainsFinalizer(op, OperationLogsFinalizer)).To(BeTrue())
			Expect(containsString(op.ObjectMeta.Finalizers, OperationLogsFinalizer)).To(BeTrue())

			controllerutil.RemoveFinalizer(op, OperationLogsFinalizer)
			Expect(len(op.GetFinalizers())).To(Equal(0))
			Expect(controllerutil.ContainsFinalizer(op, OperationLogsFinalizer)).To(BeFalse())
		})

		It("should correctly handle notifications finalizers", func() {
			op := &Job{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
				},
			}
			Expect(IsOperationBeingDeleted(op)).To(BeTrue())

			controllerutil.AddFinalizer(op, OperationStatusFinalizer)
			Expect(len(op.GetFinalizers())).To(Equal(1))
			Expect(controllerutil.ContainsFinalizer(op, OperationStatusFinalizer)).To(BeTrue())
			Expect(containsString(op.ObjectMeta.Finalizers, OperationStatusFinalizer)).To(BeTrue())

			controllerutil.RemoveFinalizer(op, OperationStatusFinalizer)
			Expect(len(op.GetFinalizers())).To(Equal(0))
			Expect(controllerutil.ContainsFinalizer(op, OperationStatusFinalizer)).To(BeFalse())
		})

		It("should correctly handle both finalizers", func() {
			op := &Job{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
				},
			}
			Expect(IsOperationBeingDeleted(op)).To(BeTrue())

			controllerutil.AddFinalizer(op, OperationLogsFinalizer)
			controllerutil.AddFinalizer(op, OperationStatusFinalizer)
			Expect(len(op.GetFinalizers())).To(Equal(2))
			Expect(controllerutil.ContainsFinalizer(op, OperationStatusFinalizer)).To(BeTrue())
			Expect(controllerutil.ContainsFinalizer(op, OperationLogsFinalizer)).To(BeTrue())

			controllerutil.RemoveFinalizer(op, OperationStatusFinalizer)
			controllerutil.RemoveFinalizer(op, OperationLogsFinalizer)
			Expect(len(op.GetFinalizers())).To(Equal(0))
			Expect(controllerutil.ContainsFinalizer(op, OperationStatusFinalizer)).To(BeFalse())
			Expect(controllerutil.ContainsFinalizer(op, OperationLogsFinalizer)).To(BeFalse())
		})

	})
})
