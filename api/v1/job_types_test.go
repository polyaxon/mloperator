package v1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// These tests are written in BDD-style using Ginkgo framework. Refer to
// http://onsi.github.io/ginkgo to learn more.
var _ = Describe("Job", func() {
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
			Expect(op.GetFinalizers()).To(HaveLen(1))
			Expect(controllerutil.ContainsFinalizer(op, OperationLogsFinalizer)).To(BeTrue())
			Expect(containsString(op.ObjectMeta.Finalizers, OperationLogsFinalizer)).To(BeTrue())

			controllerutil.RemoveFinalizer(op, OperationLogsFinalizer)
			Expect(op.GetFinalizers()).To(BeEmpty())
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
			Expect(op.GetFinalizers()).To(HaveLen(1))
			Expect(controllerutil.ContainsFinalizer(op, OperationStatusFinalizer)).To(BeTrue())
			Expect(containsString(op.ObjectMeta.Finalizers, OperationStatusFinalizer)).To(BeTrue())

			controllerutil.RemoveFinalizer(op, OperationStatusFinalizer)
			Expect(op.GetFinalizers()).To(BeEmpty())
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
			Expect(op.GetFinalizers()).To(HaveLen(2))
			Expect(controllerutil.ContainsFinalizer(op, OperationStatusFinalizer)).To(BeTrue())
			Expect(controllerutil.ContainsFinalizer(op, OperationLogsFinalizer)).To(BeTrue())

			controllerutil.RemoveFinalizer(op, OperationStatusFinalizer)
			controllerutil.RemoveFinalizer(op, OperationLogsFinalizer)
			Expect(op.GetFinalizers()).To(BeEmpty())
			Expect(controllerutil.ContainsFinalizer(op, OperationStatusFinalizer)).To(BeFalse())
			Expect(controllerutil.ContainsFinalizer(op, OperationLogsFinalizer)).To(BeFalse())
		})

	})
})
