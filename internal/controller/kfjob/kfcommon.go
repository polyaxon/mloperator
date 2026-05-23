package kfjob

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/kfjob/kfapi"
	"github.com/polyaxon/mloperator/internal/helpers/managers"
)

// isKfJobInTerminalState checks if a Kubeflow job has reached a terminal state (Succeeded or Failed)
func isKfJobInTerminalState(job *unstructured.Unstructured) bool {
	status, ok, err := unstructured.NestedFieldCopy(job.Object, "status")
	if !ok || err != nil {
		return false
	}

	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return false
	}

	jobStatus := kfapi.JobStatus{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &jobStatus)
	if err != nil || len(jobStatus.Conditions) == 0 {
		return false
	}

	// Check the last condition - must check both Type AND Status
	// Similar to IsJobSucceeded/IsJobFailed for batchv1.Job
	cond := jobStatus.Conditions[len(jobStatus.Conditions)-1]
	return (cond.Type == kfapi.JobSucceeded && cond.Status == "True") ||
		(cond.Type == kfapi.JobFailed && cond.Status == "True")
}

// Common logic for reconciling job status
func (r *KfJobReconciler) reconcileKfJobStatus(instance *apiv1.KfJob, job unstructured.Unstructured) (bool, error) {
	now := metav1.Now()
	log := r.Log

	// Check the pods
	instanceID := instance.Labels["app.kubernetes.io/instance"]
	podStatus, reason, message := managers.HasUnschedulablePods(r.Client, instanceID, instance.Namespace)
	exitStatus, err := managers.GetMainContainerExitStatusByInstance(r.Client, instanceID, instance.Namespace)
	if err != nil {
		log.Error(err, "Get main container exit status error")
	}

	logPodWarning := func() (bool, error) {
		if podStatus != apiv1.OperationWarning {
			return false, nil
		}
		log.V(1).Info("Job has unschedulable pod(s)", "Reason", reason, "message", message)
		if updated := instance.Status.LogWarning(reason, message); updated {
			log.V(1).Info("Job Logging Status Warning")
			return true, nil
		}
		return false, nil
	}

	status, ok, unerr := unstructured.NestedFieldCopy(job.Object, "status")
	if !ok {
		if unerr != nil {
			log.Error(unerr, "NestedFieldCopy unstructured to status error")
			return false, nil
		}
		log.Info("NestedFieldCopy unstructured to status error",
			"err", "Status is not found in job")
		return logPodWarning()
	}

	statusMap := status.(map[string]interface{})
	jobStatus := kfapi.JobStatus{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &jobStatus)
	if err != nil {
		log.Error(err, "Convert unstructured to status error")
		return false, err
	}

	if len(jobStatus.Conditions) == 0 {
		return logPodWarning()
	}

	cond := jobStatus.Conditions[len(jobStatus.Conditions)-1]

	if cond.Type == kfapi.JobRunning || cond.Type == kfapi.JobCreated {
		instance.Status.LogRunning()
		log.V(1).Info("Job Logging Status Running")
		return true, nil
	}

	if cond.Type == kfapi.JobSucceeded {
		instance.Status.LogSucceeded()
		instance.Status.CompletionTime = &now
		log.V(1).Info("Job Logging Status Succeeded")
		return true, nil
	}

	if cond.Type == kfapi.JobFailed {
		var attemptPtr *int32
		if exitStatus != nil {
			failedAttempts := exitStatus.ContainerFailedAttempts()
			if failedAttempts > 0 {
				attemptPtr = &failedAttempts
			}
		}
		newMessage := managers.FormatMainContainerFailureMessage(cond.Message, exitStatus, attemptPtr)
		if updated := instance.Status.LogFailed(cond.Reason, newMessage); updated {
			instance.Status.CompletionTime = &now
			log.V(1).Info("Job Logging Status Failed", "Message", newMessage, "podStatus", podStatus, "PodMessage", message)
			return true, nil
		}
	}

	if updated, err := logPodWarning(); updated || err != nil {
		return updated, err
	}

	if cond.Type == kfapi.JobRestarting {
		instance.Status.LogWarning(cond.Reason, cond.Message)
		log.V(1).Info("Job Logging Status Warning")
		return true, nil
	}
	return false, nil
}
