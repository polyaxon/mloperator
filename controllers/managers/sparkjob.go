package managers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/kinds"
	"github.com/polyaxon/mloperator/controllers/sparkapi"
)

// generateExecutorSpec generates a new ReplicaSpec
func generateExecutorSpec(replicSpec operationv1.SparkReplicaSpec, labels map[string]string) sparkapi.ExecutorSpec {
	l := make(map[string]string)
	for k, v := range labels {
		l[k] = v
	}

	return sparkapi.ExecutorSpec{}
}

// generateDriverSpec generates a new ReplicaSpec
func generateDriverSpec(replicSpec operationv1.SparkReplicaSpec, labels map[string]string) sparkapi.DriverSpec {
	l := make(map[string]string)
	for k, v := range labels {
		l[k] = v
	}

	return sparkapi.DriverSpec{}
}

// GenerateSparkJob returns a SparkJob
func GenerateSparkJob(
	name string,
	namespace string,
	labels map[string]string,
	termination operationv1.TerminationSpec,
	spec operationv1.SparkJobSpec,
) (*unstructured.Unstructured, error) {

	jobSpec := &sparkapi.SparkApplicationSpec{
		Driver:   generateDriverSpec(spec.ReplicaSpecs[operationv1.SparkReplicaTypeDriver], labels),
		Executor: generateExecutorSpec(spec.ReplicaSpecs[operationv1.SparkReplicaTypeExecutor], labels),
	}

	job := &unstructured.Unstructured{}
	job.SetAPIVersion(kinds.KFAPIVersion)
	job.SetKind(kinds.SparkApplicationKind)
	job.SetLabels(labels)
	job.SetName(name)
	job.SetNamespace(namespace)

	jobManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobSpec)

	if err != nil {
		return nil, fmt.Errorf("Convert sparkjob to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(job.Object, jobManifest, "spec"); err != nil {
		return nil, fmt.Errorf("Set .spec.hosts error: %v", err)
	}

	return job, nil
}
