package managers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster/kinds"
	"github.com/polyaxon/mloperator/internal/controller/cluster/rayapi"
	"github.com/polyaxon/mloperator/internal/helpers/utils"
)

/*
GetRayStartParams utils function to handle default case
*/
func GetRayStartParams(rayStartParams map[string]string) map[string]string {
	if rayStartParams != nil && len(rayStartParams) > 0 {
		return rayStartParams
	}
	return make(map[string]string)
}

// generateHeadGroupSpec generates a new ReplicaSpec
func generateHeadGroupSpec(replicaSpec apiv1.RayReplicaSpec, name string, labels map[string]string, annotations map[string]string) rayapi.HeadGroupSpec {
	l := make(map[string]string)
	for k, v := range replicaSpec.Template.GetLabels() {
		if k != "app.kubernetes.io/name" {
			l[k] = v
		}
	}
	for k, v := range labels {
		if k != "app.kubernetes.io/name" {
			l[k] = v
		}
	}
	a := make(map[string]string)
	for k, v := range replicaSpec.Template.GetAnnotations() {
		a[k] = v
	}
	for k, v := range annotations {
		a[k] = v
	}

	return rayapi.HeadGroupSpec{
		RayStartParams: GetRayStartParams(replicaSpec.RayStartParams),
		HeadService: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-svc", Labels: l, Annotations: a},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-svc", Labels: l, Annotations: a},
			Spec:       replicaSpec.Template.Spec,
		},
	}
}

// generateWorkerGroupSpec generates a new ReplicaSpec
func generateWorkerGroupSpec(replicaSpec apiv1.RayReplicaSpec, labels map[string]string, annotations map[string]string, idx int) rayapi.WorkerGroupSpec {
	l := make(map[string]string)
	for k, v := range labels {
		if k != "app.kubernetes.io/name" {
			l[k] = v
		}
	}
	a := make(map[string]string)
	for k, v := range annotations {
		a[k] = v
	}
	// Use groupName or generate a new name based on idx
	var groupName string
	if replicaSpec.GroupName != "" {
		groupName = replicaSpec.GroupName
	} else {
		groupName = fmt.Sprintf("worker-%d", idx)
	}
	return rayapi.WorkerGroupSpec{
		GroupName:      groupName,
		Replicas:       utils.GetNumReplicas(replicaSpec.Replicas),
		MinReplicas:    utils.GetNumReplicas(replicaSpec.MinReplicas),
		MaxReplicas:    utils.GetNumReplicas(replicaSpec.MaxReplicas),
		RayStartParams: GetRayStartParams(replicaSpec.RayStartParams),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: l, Annotations: a},
			Spec:       replicaSpec.Template.Spec,
		},
	}
}

// generateAutoscalerOptions converts our autoscaler options to rayapi.AutoscalerOptions
// It also maps termination.Culling.Timeout to IdleTimeoutSeconds when autoscaling is enabled
func generateAutoscalerOptions(opts *apiv1.RayAutoscalerOptions, termination apiv1.TerminationSpec) *rayapi.AutoscalerOptions {
	result := &rayapi.AutoscalerOptions{}
	hasConfig := false

	// Map termination.Culling.Timeout to IdleTimeoutSeconds
	if termination.Culling != nil && termination.Culling.Timeout != nil {
		result.IdleTimeoutSeconds = termination.Culling.Timeout
		hasConfig = true
	}

	if opts != nil {
		if opts.UpscalingMode != "" {
			mode := rayapi.UpscalingMode(opts.UpscalingMode)
			result.UpscalingMode = &mode
			hasConfig = true
		}

		if opts.ImagePullPolicy != "" {
			policy := corev1.PullPolicy(opts.ImagePullPolicy)
			result.ImagePullPolicy = &policy
			hasConfig = true
		}
	}

	if !hasConfig {
		return nil
	}

	return result
}

// GenerateRayCluster returns a RayCluster
func GenerateRayCluster(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	termination apiv1.TerminationSpec,
	spec apiv1.RayClusterSpec,
) (*unstructured.Unstructured, error) {
	head := generateHeadGroupSpec(spec.Head, name, labels, annotations)
	var workers []rayapi.WorkerGroupSpec
	if spec.Workers != nil && len(spec.Workers) > 0 {
		workers = make([]rayapi.WorkerGroupSpec, len(spec.Workers))
		for i, w := range spec.Workers {
			workers[i] = generateWorkerGroupSpec(w, labels, annotations, i)
		}
	} else {
		workers = nil
	}

	// Create a RayCluster spec (not RayCluster)
	clusterSpec := &rayapi.RayClusterSpec{
		RayVersion:              spec.RayVersion,
		HeadGroupSpec:           head,
		WorkerGroupSpecs:        workers,
		Suspend:                 spec.Suspend,
		EnableInTreeAutoscaling: spec.EnableInTreeAutoscaling,
		AutoscalerOptions:       generateAutoscalerOptions(spec.AutoscalerOptions, termination),
	}

	cluster := &unstructured.Unstructured{}
	cluster.SetAPIVersion(kinds.RayAPIVersion)
	cluster.SetKind(kinds.RayClusterKind)
	cluster.SetLabels(labels)
	cluster.SetAnnotations(annotations)
	cluster.SetName(name)
	cluster.SetNamespace(namespace)

	clusterManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clusterSpec)
	if err != nil {
		return nil, fmt.Errorf("Convert raycluster to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(cluster.Object, clusterManifest, "spec"); err != nil {
		return nil, fmt.Errorf("Set .spec error: %v", err)
	}

	return cluster, nil
}
