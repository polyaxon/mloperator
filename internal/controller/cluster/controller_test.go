package cluster

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/polyaxon/mloperator/internal/controller/cluster/kinds"
	"github.com/polyaxon/mloperator/internal/helpers/config"
)

var _ = Describe("Cluster Controller - Pod event mapping", func() {
	const (
		runUUID       = "525d5b803037483f8f063c8faa2fc089"
		resourceName  = "plx-operation-" + runUUID
		testNamespace = "polyaxon-agent"
	)

	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	newPod := func(name string, labels map[string]string, owners ...metav1.OwnerReference) *corev1.Pod {
		podLabels := map[string]string{
			"app.kubernetes.io/instance": runUUID,
			config.PolyaxonRunIdLabel:    runUUID,
		}
		for key, value := range labels {
			podLabels[key] = value
		}
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name,
				Namespace:       testNamespace,
				Labels:          podLabels,
				OwnerReferences: owners,
			},
		}
	}

	newControllerRef := func(gvk schema.GroupVersionKind, name string) metav1.OwnerReference {
		return *metav1.NewControllerRef(&metav1.ObjectMeta{Name: name}, gvk)
	}

	replicaSetKind := appsv1.SchemeGroupVersion.WithKind("ReplicaSet")
	expected := types.NamespacedName{Name: resourceName, Namespace: testNamespace}

	It("should map Dask scheduler pods to the Cluster name", func() {
		pod := newPod(
			resourceName+"-scheduler-6c9f8d7b4-q8w2n",
			map[string]string{
				daskClusterNameLabel: resourceName,
				"dask.org/component": "scheduler",
			},
			newControllerRef(replicaSetKind, resourceName+"-scheduler-6c9f8d7b4"),
		)

		requests := mapClusterObjectToRequest(ctx, pod)

		Expect(requests).To(HaveLen(1))
		Expect(requests[0].NamespacedName).To(Equal(expected))
	})

	It("should map Dask worker pods to the Cluster name", func() {
		pod := newPod(
			resourceName+"-default-worker-0f3c2a1b9e-5b7d9c8f6-k4m7p",
			map[string]string{
				daskClusterNameLabel:        resourceName,
				"dask.org/component":        "worker",
				"dask.org/workergroup-name": resourceName + "-default",
			},
			newControllerRef(replicaSetKind, resourceName+"-default-worker-0f3c2a1b9e-5b7d9c8f6"),
		)

		requests := mapClusterObjectToRequest(ctx, pod)

		Expect(requests).To(HaveLen(1))
		Expect(requests[0].NamespacedName).To(Equal(expected))
	})

	It("should map Ray pods to the Cluster name", func() {
		pod := newPod(
			resourceName+"-head-x7v2c",
			map[string]string{"ray.io/cluster": resourceName},
			newControllerRef(schema.FromAPIVersionAndKind(kinds.RayAPIVersion, kinds.RayClusterKind), resourceName),
		)

		requests := mapClusterObjectToRequest(ctx, pod)

		Expect(requests).To(HaveLen(1))
		Expect(requests[0].NamespacedName).To(Equal(expected))
	})

	It("should ignore pods not created by a Ray or Dask cluster", func() {
		pod := newPod(
			resourceName+"-5d8f7c9b4-x2k7q",
			nil,
			newControllerRef(replicaSetKind, resourceName+"-5d8f7c9b4"),
		)

		Expect(mapClusterObjectToRequest(ctx, pod)).To(BeEmpty())
		Expect(mapClusterObjectToRequest(ctx, newPod(resourceName, nil))).To(BeEmpty())
	})
})
