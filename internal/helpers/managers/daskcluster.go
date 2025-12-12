package managers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster/daskapi"
	"github.com/polyaxon/mloperator/internal/controller/cluster/kinds"
)

// generateHeadGroupSpec generates a new ReplicaSpec
func generateClusterSpec(worker apiv1.DaskReplicaSpec, scheduler apiv1.DaskReplicaSpec, service corev1.ServiceSpec, labels map[string]string, annotations map[string]string) daskapi.DaskCluster {
	// Merge labels from input labels and template labels
	workerLabels := make(map[string]string)
	for k, v := range labels {
		workerLabels[k] = v
	}
	for k, v := range worker.Template.GetLabels() {
		workerLabels[k] = v
	}

	schedulerLabels := make(map[string]string)
	for k, v := range labels {
		schedulerLabels[k] = v
	}
	for k, v := range scheduler.Template.GetLabels() {
		schedulerLabels[k] = v
	}

	// Merge annotations from input annotations and template annotations
	workerAnnotations := make(map[string]string)
	for k, v := range annotations {
		workerAnnotations[k] = v
	}
	for k, v := range worker.Template.GetAnnotations() {
		workerAnnotations[k] = v
	}

	schedulerAnnotations := make(map[string]string)
	for k, v := range annotations {
		schedulerAnnotations[k] = v
	}
	for k, v := range scheduler.Template.GetAnnotations() {
		schedulerAnnotations[k] = v
	}

	return daskapi.DaskCluster{
		Spec: daskapi.DaskClusterSpec{
			Worker: daskapi.WorkerSpec{
				Metadata: &metav1.ObjectMeta{
					Labels:      workerLabels,
					Annotations: workerAnnotations,
				},
				Replicas: worker.Replicas,
				Spec:     worker.Template.Spec,
			},
			Scheduler: daskapi.SchedulerSpec{
				Metadata: &metav1.ObjectMeta{
					Labels:      schedulerLabels,
					Annotations: schedulerAnnotations,
				},
				Spec:    scheduler.Template.Spec,
				Service: service,
			},
		},
	}
}

// GenerateDaskCluster returns a DaskCluster resource for the Dask operator
func GenerateDaskCluster(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	termination apiv1.TerminationSpec,
	spec apiv1.DaskClusterSpec,
) (*unstructured.Unstructured, error) {
	cluster := generateClusterSpec(spec.ReplicaSpecs[apiv1.DaskReplicaTypeWorker], spec.ReplicaSpecs[apiv1.DaskReplicaTypeScheduler], spec.Service, labels, annotations)

	clusterResource := &unstructured.Unstructured{}
	clusterResource.SetAPIVersion(kinds.DaskAPIVersion)
	clusterResource.SetKind(kinds.DaskClusterKind)
	clusterResource.SetLabels(labels)
	clusterResource.SetAnnotations(annotations)
	clusterResource.SetName(name)
	clusterResource.SetNamespace(namespace)

	clusterManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cluster.Spec)

	if err != nil {
		return nil, fmt.Errorf("convert DaskCluster to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(clusterResource.Object, clusterManifest, "spec"); err != nil {
		return nil, fmt.Errorf("set .spec error: %v", err)
	}

	return clusterResource, nil
}

// GenerateDaskAutoscaler returns a DaskAutoscaler resource for the Dask operator
func GenerateDaskAutoscaler(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	spec apiv1.DaskClusterSpec,
) (*unstructured.Unstructured, error) {
	// Only create autoscaler if both min and max replicas are set
	if spec.MinReplicas == nil || spec.MaxReplicas == nil {
		return nil, nil
	}

	autoscaler := daskapi.DaskAutoscaler{
		Spec: daskapi.DaskAutoscalerSpec{
			Cluster: name,
			Minimum: int(*spec.MinReplicas),
			Maximum: int(*spec.MaxReplicas),
		},
	}

	autoscalerResource := &unstructured.Unstructured{}
	autoscalerResource.SetAPIVersion(kinds.DaskAPIVersion)
	autoscalerResource.SetKind(kinds.DaskAutoscalerKind)
	autoscalerResource.SetLabels(labels)
	autoscalerResource.SetAnnotations(annotations)
	autoscalerResource.SetName(name)
	autoscalerResource.SetNamespace(namespace)

	autoscalerManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&autoscaler.Spec)

	if err != nil {
		return nil, fmt.Errorf("convert DaskAutoscaler to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(autoscalerResource.Object, autoscalerManifest, "spec"); err != nil {
		return nil, fmt.Errorf("set .spec error: %v", err)
	}

	return autoscalerResource, nil
}
