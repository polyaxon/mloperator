package managers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/kfapi"
	"github.com/polyaxon/mloperator/controllers/kinds"
	"github.com/polyaxon/mloperator/controllers/utils"
)

// GenerateXGBJob returns a XGBJob
func GenerateXGBJob(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	termination operationv1.TerminationSpec,
	spec operationv1.XGBoostJobSpec,
) (*unstructured.Unstructured, error) {
	replicaSpecs := map[operationv1.XGBReplicaType]*operationv1.KFReplicaSpec{}
	for k, v := range spec.ReplicaSpecs {
		replicaSpecs[operationv1.XGBReplicaType(k)] = generateKFReplica(*v, labels, annotations)
	}

	jobSpec := &kfapi.XGBoostJobSpec{
		RunPolicy: kfapi.RunPolicy{
			ActiveDeadlineSeconds:   termination.ActiveDeadlineSeconds,
			BackoffLimit:            utils.GetBackoffLimit(termination.BackoffLimit),
			TTLSecondsAfterFinished: utils.GetTTL(termination.TTLSecondsAfterFinished),
			CleanPodPolicy:          spec.CleanPodPolicy,
			SchedulingPolicy:        spec.SchedulingPolicy,
		},
		XGBReplicaSpecs: replicaSpecs,
	}

	job := &unstructured.Unstructured{}
	job.SetAPIVersion(kinds.KFAPIVersion)
	job.SetKind(kinds.XGBoostJobKind)
	job.SetLabels(labels)
	job.SetName(name)
	job.SetNamespace(namespace)

	jobManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobSpec)

	if err != nil {
		return nil, fmt.Errorf("Convert xgbjob to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(job.Object, jobManifest, "spec"); err != nil {
		return nil, fmt.Errorf("Set .spec.hosts error: %v", err)
	}

	return job, nil
}
