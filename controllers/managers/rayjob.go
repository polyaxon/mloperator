package managers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/kinds"
	"github.com/polyaxon/mloperator/controllers/rayapi"
	"github.com/polyaxon/mloperator/controllers/utils"
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
func generateHeadGroupSpec(replicaSpec operationv1.RayReplicaSpec, name string, labels map[string]string, annotations map[string]string) rayapi.HeadGroupSpec {
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
func generateWorkerGroupSpec(replicaSpec operationv1.RayReplicaSpec, labels map[string]string, annotations map[string]string, idx int) rayapi.WorkerGroupSpec {
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

// GenerateRayJob returns a RayJob
func GenerateRayJob(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	termination operationv1.TerminationSpec,
	spec operationv1.RayJobSpec,
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

	cluster := &rayapi.RayClusterSpec{
		RayVersion:       spec.RayVersion,
		HeadGroupSpec:    head,
		WorkerGroupSpecs: workers,
	}

	var activeDeadlineSeconds *int32

	if termination.ActiveDeadlineSeconds != nil {
		value := int32(*termination.ActiveDeadlineSeconds)
		activeDeadlineSeconds = &value
	} else {
		activeDeadlineSeconds = nil
	}
	jobSpec := &rayapi.RayJobSpec{
		Entrypoint:               spec.Entrypoint,
		Metadata:                 spec.Metadata,
		RuntimeEnvYAML:           spec.RuntimeEnv,
		JobId:                    name,
		ShutdownAfterJobFinishes: true,
		ActiveDeadlineSeconds:    activeDeadlineSeconds,
		TTLSecondsAfterFinished:  utils.GetTTL(termination.TTLSecondsAfterFinished),
		RayClusterSpec:           cluster,
		SubmissionMode:           "HTTPMode",
	}
	jobStatus := &rayapi.RayJobStatus{
		JobId:          name,
		RayClusterName: name,
	}

	job := &unstructured.Unstructured{}
	job.SetAPIVersion(kinds.RayAPIVersion)
	job.SetKind(kinds.RayJobKind)
	job.SetLabels(labels)
	job.SetAnnotations(annotations)
	job.SetName(name)
	job.SetNamespace(namespace)

	jobManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobSpec)
	jobStatusManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobStatus)

	if err != nil {
		return nil, fmt.Errorf("Convert rayjob to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(job.Object, jobManifest, "spec"); err != nil {
		return nil, fmt.Errorf("Set .spec error: %v", err)
	}
	if err := unstructured.SetNestedField(job.Object, jobStatusManifest, "status"); err != nil {
		return nil, fmt.Errorf("Set status error: %v", err)
	}

	return job, nil
}
