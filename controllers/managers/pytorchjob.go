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

// GeneratePytorchJob returns a PytorchJob
func GeneratePytorchJob(
	name string,
	namespace string,
	labels map[string]string,
	termination operationv1.TerminationSpec,
	spec operationv1.PytorchJobSpec,
) (*unstructured.Unstructured, error) {
	replicaSpecs := map[operationv1.PyTorchReplicaType]*operationv1.KFReplicaSpec{}
	for k, v := range spec.ReplicaSpecs {
		replicaSpecs[operationv1.PyTorchReplicaType(k)] = generateKFReplica(v)
	}

	// copy all of the labels to the pod including pod default related labels
	for _, replicaSpec := range replicaSpecs {
		l := &replicaSpec.Template.ObjectMeta.Labels
		for k, v := range labels {
			(*l)[k] = v
		}
	}

	jobSpec := &kfapi.PyTorchJobSpec{
		RunPolicy: kfapi.RunPolicy{
			ActiveDeadlineSeconds:   termination.ActiveDeadlineSeconds,
			BackoffLimit:            utils.GetBackoffLimit(termination.BackoffLimit),
			TTLSecondsAfterFinished: utils.GetTTL(termination.TTLSecondsAfterFinished),
			CleanPodPolicy:          spec.CleanPodPolicy,
			SchedulingPolicy:        spec.SchedulingPolicy,
		},
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
		return nil, fmt.Errorf("Convert pytorchjob to unstructured error: %v", err)
	}

	if err := unstructured.SetNestedField(job.Object, jobManifest, "spec"); err != nil {
		return nil, fmt.Errorf("Set .spec.hosts error: %v", err)
	}

	return job, nil
}
