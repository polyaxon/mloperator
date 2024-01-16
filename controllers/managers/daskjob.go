package managers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/daskapi"
	"github.com/polyaxon/mloperator/controllers/kinds"
)

// generateHeadGroupSpec generates a new ReplicaSpec
func generateClusterSpec(worker operationv1.DaskReplicaSpec, scheduler operationv1.DaskReplicaSpec, service corev1.ServiceSpec, labels map[string]string, annotations map[string]string) daskapi.DaskCluster {
	l := make(map[string]string)
	for k, v := range labels {
		l[k] = v
	}
	a := make(map[string]string)
	for k, v := range annotations {
		a[k] = v
	}

	return daskapi.DaskCluster{
		Spec: daskapi.DaskClusterSpec{
			Worker: daskapi.WorkerSpec{
				Replicas: worker.Replicas,
				Spec:     worker.Template.Spec,
			},
			Scheduler: daskapi.SchedulerSpec{
				Spec:    scheduler.Template.Spec,
				Service: service,
			},
		},
	}
}

// GenerateDaskJob returns a DaskJob
func GenerateDaskJob(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	termination operationv1.TerminationSpec,
	spec operationv1.DaskJobSpec,
) (*unstructured.Unstructured, error) {
	cluster := generateClusterSpec(spec.ReplicaSpecs[operationv1.DaskReplicaTypeWorker], spec.ReplicaSpecs[operationv1.DaskReplicaTypeScheduler], spec.Service, labels, annotations)

	jobSpec := &daskapi.DaskJobSpec{
		Job: daskapi.JobSpec{
			Spec: spec.ReplicaSpecs[operationv1.DaskReplicaTypeJob].Template.Spec,
		},
		Cluster: cluster,
	}

	job := &unstructured.Unstructured{}
	job.SetAPIVersion(kinds.DaskAPIVersion)
	job.SetKind(kinds.DaskJobKind)
	job.SetLabels(labels)
	job.SetAnnotations(annotations)
	job.SetName(name)
	job.SetNamespace(namespace)

	jobManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobSpec)

	if err != nil {
		return nil, fmt.Errorf("Convert daskjob to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(job.Object, jobManifest, "spec"); err != nil {
		return nil, fmt.Errorf("Set .spec.hosts error: %v", err)
	}

	return job, nil
}
