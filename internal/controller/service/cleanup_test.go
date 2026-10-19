package service

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiv1 "github.com/polyaxon/mloperator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Service Controller - Active deadline", func() {
	var (
		ctx        context.Context
		reconciler *ServiceReconciler
	)

	exited := corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0, Reason: "Completed"}}
	invalidImage := corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "InvalidImageName", Message: `couldn't parse image name "Busybox:latest"`}}

	// newService returns a service with a one hour deadline, started the given duration ago
	newService := func(started time.Duration, conditions ...apiv1.OperationConditionType) *apiv1.Service {
		deadline := int64(3600)
		startTime := metav1.NewTime(time.Now().Add(-started))
		instance := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "plx-operation-deadline",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/instance": "plx-operation-deadline"},
			},
			Termination: apiv1.TerminationSpec{ActiveDeadlineSeconds: &deadline},
			ServiceSpec: &apiv1.ServiceSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
				},
			},
			Status: apiv1.OperationStatus{StartTime: &startTime},
		}
		for _, condType := range conditions {
			instance.Status.Conditions = append(instance.Status.Conditions, apiv1.NewOperationCondition(condType, corev1.ConditionTrue, "test", "test"))
		}
		return instance
	}

	// withCulling enables culling with a probe that fails without the owner and project annotations, so culling rechecks after its timeout
	withCulling := func(instance *apiv1.Service) {
		timeout := int32(60)
		instance.Termination.Culling = &apiv1.CullingSpec{Timeout: &timeout}
		instance.Termination.Probe = &apiv1.ActivityProbe{Http: &apiv1.ActivityProbeHttp{Port: 8888}}
	}

	newPod := func(instance *apiv1.Service, state corev1.ContainerState) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: instance.Name + "-pod", Namespace: instance.Namespace, Labels: instance.Labels},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{
				Phase:             corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{{Name: "main", State: state}},
			},
		}
	}

	// setFakeClient stores the service with its Deployment, so the reconcile doesn't create it and log Starting
	setFakeClient := func(instance *apiv1.Service, objects ...client.Object) {
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: instance.Name, Namespace: instance.Namespace}}
		Expect(ctrl.SetControllerReference(instance, deployment, k8sClient.Scheme())).To(Succeed())
		reconciler = &ServiceReconciler{
			Client: fake.NewClientBuilder().
				WithScheme(k8sClient.Scheme()).
				WithObjects(append(objects, instance, deployment)...).
				WithStatusSubresource(&apiv1.Service{}).
				Build(),
			Log:    ctrl.Log.WithName("controllers").WithName("Service"),
			Scheme: k8sClient.Scheme(),
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should check culling for a running service within its deadline", func() {
		instance := newService(time.Minute, apiv1.OperationStarting, apiv1.OperationRunning)
		withCulling(instance)
		setFakeClient(instance)

		result, err := reconciler.reconcileService(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(time.Minute))
	})

	It("should requeue at the deadline when it comes before the next culling check", func() {
		instance := newService(time.Hour-30*time.Second, apiv1.OperationStarting, apiv1.OperationRunning)
		withCulling(instance)
		setFakeClient(instance)

		result, err := reconciler.reconcileService(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeNumerically("~", 30*time.Second, 5*time.Second))
	})

	It("should mark a service within its deadline succeeded when its main container exits", func() {
		instance := newService(time.Minute, apiv1.OperationStarting, apiv1.OperationRunning)
		setFakeClient(instance, newPod(instance, exited))

		_, err := reconciler.reconcileService(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		Expect(instance.Status.IsSucceeded()).To(BeTrue())
	})

	It("should requeue at the deadline after a warning", func() {
		instance := newService(time.Minute, apiv1.OperationStarting, apiv1.OperationRunning, apiv1.OperationWarning)
		setFakeClient(instance)

		result, err := reconciler.reconcileService(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeNumerically("~", 59*time.Minute, time.Minute))
	})

	It("should keep a service that fails past its deadline as failed", func() {
		instance := newService(2*time.Hour, apiv1.OperationStarting, apiv1.OperationRunning)
		setFakeClient(instance, newPod(instance, invalidImage))

		_, err := reconciler.reconcileService(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		persisted := &apiv1.Service{}
		Expect(reconciler.Get(ctx, client.ObjectKeyFromObject(instance), persisted)).To(Succeed())
		condition := persisted.Status.Conditions[len(persisted.Status.Conditions)-1]
		Expect(condition.Type).To(Equal(apiv1.OperationFailed))
		Expect(condition.Reason).To(Equal("InvalidImageName"))
		Expect(persisted.Status.IsStopped()).To(BeFalse())
	})

	It("should stop and delete a service past its deadline even when its main container exited", func() {
		instance := newService(2*time.Hour, apiv1.OperationStarting, apiv1.OperationRunning)
		setFakeClient(instance, newPod(instance, exited))

		_, err := reconciler.reconcileService(ctx, instance)

		Expect(err).NotTo(HaveOccurred())
		condition := instance.Status.Conditions[len(instance.Status.Conditions)-1]
		Expect(condition.Type).To(Equal(apiv1.OperationStopped))
		Expect(condition.Reason).To(Equal("DeadlineExceeded"))
		Expect(instance.Status.IsSucceeded()).To(BeFalse())
		err = reconciler.Get(ctx, client.ObjectKeyFromObject(instance), &apiv1.Service{})
		Expect(apierrs.IsNotFound(err)).To(BeTrue())
	})
})
