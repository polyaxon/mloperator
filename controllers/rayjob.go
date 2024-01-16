package controllers

import (
	"context"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/kinds"
	"github.com/polyaxon/mloperator/controllers/managers"
	"github.com/polyaxon/mloperator/controllers/rayapi"
)

func (r *OperationReconciler) reconcileRayJobOp(ctx context.Context, instance *operationv1.Operation) (ctrl.Result, error) {
	// Reconcile the underlaying job
	return ctrl.Result{}, r.reconcileRayJob(ctx, instance)
}

func (r *OperationReconciler) reconcileRayJob(ctx context.Context, instance *operationv1.Operation) error {
	log := r.Log

	job, err := managers.GenerateRayJob(
		instance.Name,
		instance.Namespace,
		instance.Labels,
		instance.Annotations,
		instance.Termination,
		*instance.RayJobSpec,
	)

	if err != nil {
		log.V(1).Info("GenerateRayJob Error")
		return err
	}

	if err := ctrl.SetControllerReference(instance, job, r.Scheme); err != nil {
		log.V(1).Info("SetControllerReference Error")
		return err
	}

	// Check if the Job already exists
	foundJob := &unstructured.Unstructured{}
	foundJob.SetAPIVersion(kinds.RayAPIVersion)
	foundJob.SetKind(kinds.RayJobKind)
	justCreated := false
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundJob)
	if err != nil && apierrs.IsNotFound(err) {
		if instance.IsDone() {
			return nil
		}
		log.V(1).Info("Creating RayJob", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Create(ctx, job)
		if err != nil {
			if updated := instance.LogWarning("OperatorCreateRayJob", err.Error()); updated {
				log.V(1).Info("Warning unable to create RayJob")
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				r.instanceSyncStatus(instance)
			}
			return err
		}
		justCreated = true
		instance.LogStarting()
		err = r.Status().Update(ctx, instance)
		r.instanceSyncStatus(instance)
	} else if err != nil {
		return err
	}

	// Update the job object and write the result back if there are any changes
	if !justCreated && !instance.IsDone() && managers.CopyKFJobFields(job, foundJob) {
		log.V(1).Info("Updating RayJob", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Update(ctx, foundJob)
		if err != nil {
			return err
		}
	}

	// Check the job status
	condUpdated, err := r.reconcileRayJobStatus(instance, *foundJob)
	if err != nil {
		log.V(1).Info("reconcileRayJobStatus Error")
		return err
	}
	if condUpdated {
		log.V(1).Info("Reconciling PyTorchJob status", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Status().Update(ctx, instance)
		if err != nil {
			return err
		}
		r.instanceSyncStatus(instance)
	}

	return nil
}

func (r *OperationReconciler) reconcileRayJobStatus(instance *operationv1.Operation, job unstructured.Unstructured) (bool, error) {
	now := metav1.Now()
	log := r.Log

	// Check the pods
	podStatus, reason, message := managers.HasUnschedulablePods(r.Client, instance)
	if podStatus == operationv1.OperationWarning {
		log.V(1).Info("Service has unschedulable pod(s)", "Reason", reason, "message", message)
		if updated := instance.LogWarning(reason, message); updated {
			log.V(1).Info("Service Logging Status Warning")
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
		return false, nil
	}

	statusMap := status.(map[string]interface{})
	jobStatus := rayapi.RayJobStatus{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &jobStatus)
	if err != nil {
		log.Error(err, "Convert unstructured to status error")
		return false, err
	}

	if jobStatus.JobStatus == rayapi.JobStatusRunning {
		instance.LogRunning()
		log.V(1).Info("Job Logging Status Running")
		return true, nil
	}

	if jobStatus.JobStatus == rayapi.JobStatusSucceeded {
		instance.LogSucceeded()
		instance.Status.CompletionTime = &now
		log.V(1).Info("Job Logging Status Succeeded")
		return true, nil
	}

	if jobStatus.JobStatus == rayapi.JobStatusFailed {
		newMessage := operationv1.GetFailureMessage(jobStatus.Message, podStatus, reason, message)
		if updated := instance.LogFailed(reason, newMessage); updated {
			instance.Status.CompletionTime = &now
			log.V(1).Info("Job Logging Status Failed", "Message", newMessage, "podStatus", podStatus, "PodMessage", message)
			return true, nil
		}
	}

	if jobStatus.JobStatus == rayapi.JobStatusStopped {
		newMessage := operationv1.GetFailureMessage(jobStatus.Message, podStatus, reason, message)
		if updated := instance.LogStopped(jobStatus.RayClusterStatus.Reason, newMessage); updated {
			instance.Status.CompletionTime = &now
			log.V(1).Info("Job Logging Status Stopped", "Message", newMessage, "podStatus", podStatus, "PodMessage", message)
			return true, nil
		}
	}

	if jobStatus.JobStatus == rayapi.JobStatusPending {
		instance.LogWarning(jobStatus.RayClusterStatus.Reason, jobStatus.Message)
		log.V(1).Info("Job Logging Status Warning")
		return true, nil
	}
	return false, nil
}

func (r *OperationReconciler) cleanUpRayJob(ctx context.Context, instance *operationv1.Operation) (ctrl.Result, error) {
	return r.handleTTL(ctx, instance)
}
