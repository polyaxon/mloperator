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

// generateHeadGroupSpec generates a new ReplicaSpec
func generateHeadGroupSpec(replicSpec operationv1.RayReplicaSpec, labels map[string]string) rayapi.HeadGroupSpec {
	l := make(map[string]string)
	for k, v := range labels {
		l[k] = v
	}

	return rayapi.HeadGroupSpec{
		RayStartParams: replicSpec.RayStartParams,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: l},
			Spec:       replicSpec.Template.Spec,
		},
	}
}

// generateWorkerGroupSpec generates a new ReplicaSpec
func generateWorkerGroupSpec(replicSpec operationv1.RayReplicaSpec, labels map[string]string) rayapi.WorkerGroupSpec {
	l := make(map[string]string)
	for k, v := range labels {
		l[k] = v
	}
	return rayapi.WorkerGroupSpec{
		GroupName:      replicSpec.GroupName,
		Replicas:       replicSpec.Replicas,
		MinReplicas:    replicSpec.MinReplicas,
		MaxReplicas:    replicSpec.MaxReplicas,
		RayStartParams: replicSpec.RayStartParams,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: l},
			Spec:       replicSpec.Template.Spec,
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
	head := generateHeadGroupSpec(spec.Head, labels)
	workers := []rayapi.WorkerGroupSpec{}
	for i, w := range spec.Workers {
		workers[i] = generateWorkerGroupSpec(w, labels)
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
