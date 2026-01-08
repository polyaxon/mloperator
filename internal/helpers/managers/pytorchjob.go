package managers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/kfjob/kfapi"
	"github.com/polyaxon/mloperator/internal/controller/kfjob/kinds"
	"github.com/polyaxon/mloperator/internal/helpers/utils"
)

// GeneratePytorchJob returns a PytorchJob
func GeneratePytorchJob(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	termination apiv1.TerminationSpec,
	spec apiv1.PytorchJobSpec,
) (*unstructured.Unstructured, error) {
	replicaSpecs := map[apiv1.PyTorchReplicaType]*apiv1.KFReplicaSpec{}
	for k, v := range spec.ReplicaSpecs {
		replicaSpecs[apiv1.PyTorchReplicaType(k)] = generateKFReplica(*v, labels, annotations)
	}

	jobSpec := &kfapi.PyTorchJobSpec{
		RunPolicy: kfapi.RunPolicy{
			ActiveDeadlineSeconds:   termination.ActiveDeadlineSeconds,
			BackoffLimit:            utils.GetBackoffLimit(termination.BackoffLimit),
			TTLSecondsAfterFinished: utils.GetTTL(termination.TTLSecondsAfterFinished),
			CleanPodPolicy:          spec.CleanPodPolicy,
			SchedulingPolicy:        spec.SchedulingPolicy,
		},
		ElasticPolicy:       spec.ElasticPolicy,
		PyTorchReplicaSpecs: replicaSpecs,
	}

	job := &unstructured.Unstructured{}
	job.SetAPIVersion(kinds.KFAPIVersion)
	job.SetKind(kinds.PytorchJobKind)
	job.SetLabels(labels)
	job.SetName(name)
	job.SetNamespace(namespace)

	jobManifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobSpec)

	if err != nil {
		return nil, fmt.Errorf("convert pytorchjob to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(job.Object, jobManifest, "spec"); err != nil {
		return nil, fmt.Errorf("set .spec.hosts error: %v", err)
	}

	return job, nil
}
