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
func generateHeadGroupSpec(replicaSpec operationv1.RayReplicaSpec, name string, labels map[string]string) rayapi.HeadGroupSpec {
	l := make(map[string]string)
	for k, v := range labels {
		if k != "app.kubernetes.io/name" {
			l[k] = v
		}
	}

	return rayapi.HeadGroupSpec{
		RayStartParams: GetRayStartParams(replicaSpec.RayStartParams),
		HeadService: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Labels: l},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: l},
			Spec:       replicaSpec.Template.Spec,
		},
	}
}

// generateWorkerGroupSpec generates a new ReplicaSpec
func generateWorkerGroupSpec(replicaSpec operationv1.RayReplicaSpec, labels map[string]string, idx int) rayapi.WorkerGroupSpec {
	l := make(map[string]string)
	for k, v := range labels {
		if k != "app.kubernetes.io/name" {
			l[k] = v
		}
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
			ObjectMeta: metav1.ObjectMeta{Labels: l},
			Spec:       replicaSpec.Template.Spec,
		},
	}
}

// GenerateRayJob returns a RayJob
func GenerateRayJob(
	name string,
	namespace string,
	labels map[string]string,
	termination operationv1.TerminationSpec,
	spec operationv1.RayJobSpec,
) (*unstructured.Unstructured, error) {
	head := generateHeadGroupSpec(spec.Head, name, labels)
	var workers []rayapi.WorkerGroupSpec
	if spec.Workers != nil && len(spec.Workers) > 0 {
		workers = make([]rayapi.WorkerGroupSpec, len(spec.Workers))
		for i, w := range spec.Workers {
			workers[i] = generateWorkerGroupSpec(w, labels, i)
		}
	} else {
		workers = nil
	}

	cluster := &rayapi.RayClusterSpec{
		RayVersion:       spec.RayVersion,
		HeadGroupSpec:    head,
		WorkerGroupSpecs: workers,
	}

	// TODO: Replace shutdownAfterJobFinishes with termination.ActiveDeadlineSeconds
	jobSpec := &rayapi.RayJobSpec{
		Entrypoint:               spec.Entrypoint,
		Metadata:                 spec.Metadata,
		RuntimeEnv:               spec.RuntimeEnv,
		JobId:                    name,
		ShutdownAfterJobFinishes: true,
		TTLSecondsAfterFinished:  utils.GetTTL(termination.TTLSecondsAfterFinished),
		RayClusterSpec:           cluster,
	}

	job := &unstructured.Unstructured{}
	job.SetAPIVersion(kinds.RayAPIVersion)
	job.SetKind(kinds.RayJobKind)
	job.SetLabels(labels)
	job.SetName(name)
	job.SetNamespace(namespace)

	jobManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobSpec)

	if err != nil {
		return nil, fmt.Errorf("Convert rayjob to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(job.Object, jobManifest, "spec"); err != nil {
		return nil, fmt.Errorf("Set .spec.hosts error: %v", err)
	}

	return job, nil
}
