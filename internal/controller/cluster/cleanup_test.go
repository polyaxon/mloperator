package cluster

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
)

var _ = Describe("Cluster Controller - Active deadline", func() {
	var (
		ctx        context.Context
		reconciler *ClusterReconciler
	)

	// newCluster returns a cluster with a one hour deadline, started the given duration ago
	newCluster := func(started time.Duration, conditions ...apiv1.OperationConditionType) *apiv1.Cluster {
		deadline := int64(3600)
		startTime := metav1.NewTime(time.Now().Add(-started))
		instance := &apiv1.Cluster{
			ObjectMeta:  metav1.ObjectMeta{Name: "plx-operation-deadline", Namespace: "default"},
			Termination: apiv1.TerminationSpec{ActiveDeadlineSeconds: &deadline},
			Status:      apiv1.OperationStatus{StartTime: &startTime},
		}
		for _, condType := range conditions {
			instance.Status.Conditions = append(instance.Status.Conditions, apiv1.NewOperationCondition(condType, corev1.ConditionTrue, "test", "test"))
		}
		return instance
	}

	setFakeClient := func(instance *apiv1.Cluster) {
		scheme := runtime.NewScheme()
		Expect(apiv1.AddToScheme(scheme)).To(Succeed())
		reconciler = &ClusterReconciler{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance).WithStatusSubresource(&apiv1.Cluster{}).Build(),
			Log:    zap.New(zap.UseDevMode(true)),
			Scheme: scheme,
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should stop and delete a cluster past its deadline", func() {
		instance := newCluster(2*time.Hour, apiv1.OperationStarting, apiv1.OperationRunning)
		setFakeClient(instance)

		duration, err := reconciler.handlePastActiveDeadline(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		Expect(duration).To(BeNil())
		condition := instance.Status.Conditions[len(instance.Status.Conditions)-1]
		Expect(condition.Type).To(Equal(apiv1.OperationStopped))
		Expect(condition.Reason).To(Equal("DeadlineExceeded"))
		err = reconciler.Get(ctx, client.ObjectKeyFromObject(instance), &apiv1.Cluster{})
		Expect(apierrs.IsNotFound(err)).To(BeTrue())
	})

	It("should leave a done cluster past its deadline as it is", func() {
		instance := newCluster(2*time.Hour, apiv1.OperationStarting, apiv1.OperationRunning, apiv1.OperationSucceeded)
		setFakeClient(instance)

		duration, err := reconciler.handlePastActiveDeadline(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		Expect(duration).To(BeNil())
		Expect(instance.Status.IsSucceeded()).To(BeTrue())
		Expect(instance.Status.IsStopped()).To(BeFalse())
		Expect(reconciler.Get(ctx, client.ObjectKeyFromObject(instance), &apiv1.Cluster{})).To(Succeed())
	})

	It("should requeue at the deadline after a warning", func() {
		instance := newCluster(time.Minute, apiv1.OperationStarting, apiv1.OperationRunning, apiv1.OperationWarning)
		setFakeClient(instance)

		duration, err := reconciler.handlePastActiveDeadline(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		Expect(duration).NotTo(BeNil())
		Expect(*duration).To(BeNumerically("~", 59*time.Minute, time.Minute))
	})
})
