package cluster

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster/daskapi"
	"github.com/polyaxon/mloperator/internal/controller/cluster/kinds"
)

var _ = Describe("Cluster Controller - Dask status", func() {
	const (
		name          = "plx-operation-dask"
		testNamespace = "default"
		quotaMessage  = `pods "plx-operation-dask-default-worker-6c9f8d7b4-q8w2n" is forbidden: exceeded quota: dask-quota`
	)

	var (
		ctx        context.Context
		reconciler *ClusterReconciler
		instance   *apiv1.Cluster
		deployment *appsv1.Deployment
		cluster    unstructured.Unstructured
	)

	running := corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}

	// newPod returns a pod of the given Dask component whose main container is in the given state
	newPod := func(component string, state corev1.ContainerState) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name + "-" + component,
				Namespace: testNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/instance": name,
					daskClusterNameLabel:         name,
					daskComponentLabel:           component,
				},
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: component}}},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{Name: component, State: state}},
			},
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		instance = &apiv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
				Labels:    map[string]string{"app.kubernetes.io/instance": name},
			},
		}
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name + "-default-worker",
				Namespace: testNamespace,
				Labels:    map[string]string{daskClusterNameLabel: name},
			},
			Status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{
						Type:    appsv1.DeploymentReplicaFailure,
						Status:  corev1.ConditionTrue,
						Reason:  "FailedCreate",
						Message: quotaMessage,
					},
				},
			},
		}
		// The Dask operator reports Running as soon as the scheduler Service exists
		cluster = unstructured.Unstructured{Object: map[string]interface{}{
			"status": map[string]interface{}{"phase": string(daskapi.DaskClusterRunning)},
		}}
		cluster.SetAPIVersion(kinds.DaskAPIVersion)
		cluster.SetKind(kinds.DaskClusterKind)
		cluster.SetName(name)
		cluster.SetNamespace(testNamespace)

		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(appsv1.AddToScheme(scheme)).To(Succeed())
		Expect(apiv1.AddToScheme(scheme)).To(Succeed())
		reconciler = &ClusterReconciler{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance, deployment).Build(),
			Log:    zap.New(zap.UseDevMode(true)),
			Scheme: scheme,
		}
	})

	It("should keep a running cluster in Warning while a Deployment can't create pods", func() {
		updated, err := reconciler.reconcileDaskClusterStatus(instance, cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeTrue())
		condition := instance.Status.Conditions[len(instance.Status.Conditions)-1]
		Expect(condition.Type).To(Equal(apiv1.OperationWarning))
		Expect(condition.Reason).To(Equal("FailedCreate"))
		Expect(condition.Message).To(Equal(quotaMessage))

		updated, err = reconciler.reconcileDaskClusterStatus(instance, cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeFalse())
		Expect(instance.Status.IsWarning()).To(BeTrue())
	})

	It("should report Running once the Deployment recovers", func() {
		Expect(reconciler.Create(ctx, newPod("scheduler", running))).To(Succeed())
		_, err := reconciler.reconcileDaskClusterStatus(instance, cluster)
		Expect(err).NotTo(HaveOccurred())

		current := &appsv1.Deployment{}
		Expect(reconciler.Get(ctx, client.ObjectKeyFromObject(deployment), current)).To(Succeed())
		current.Status.Conditions = nil
		Expect(reconciler.Status().Update(ctx, current)).To(Succeed())

		updated, err := reconciler.reconcileDaskClusterStatus(instance, cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeTrue())
		Expect(instance.Status.IsRunning()).To(BeTrue())
	})

	It("should report Running only once the scheduler starts", func() {
		instance.Status.LogStarting()
		Expect(reconciler.Delete(ctx, deployment)).To(Succeed())
		Expect(reconciler.Create(ctx, newPod("worker", running))).To(Succeed())
		scheduler := newPod("scheduler", corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}})
		Expect(reconciler.Create(ctx, scheduler)).To(Succeed())

		updated, err := reconciler.reconcileDaskClusterStatus(instance, cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeFalse())
		Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).To(Equal(apiv1.OperationStarting))

		scheduler.Status.ContainerStatuses[0].State = running
		Expect(reconciler.Status().Update(ctx, scheduler)).To(Succeed())

		updated, err = reconciler.reconcileDaskClusterStatus(instance, cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeTrue())
		Expect(instance.Status.IsRunning()).To(BeTrue())
	})
})
