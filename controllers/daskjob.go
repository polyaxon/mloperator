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
	"github.com/polyaxon/mloperator/controllers/daskapi"
	"github.com/polyaxon/mloperator/controllers/kinds"
	"github.com/polyaxon/mloperator/controllers/managers"
)

func (r *OperationReconciler) reconcileDaskJobOp(ctx context.Context, instance *operationv1.Operation) (ctrl.Result, error) {
	// Reconcile the underlaying job
	return ctrl.Result{}, r.reconcileDaskJob(ctx, instance)
}

func (r *OperationReconciler) reconcileDaskJob(ctx context.Context, instance *operationv1.Operation) error {
	log := r.Log

	job, err := managers.GenerateDaskJob(
		instance.Name,
		instance.Namespace,
		instance.Labels,
		instance.Termination,
		*instance.DaskJobSpec,
	)

	if err != nil {
		log.V(1).Info("GenerateDaskJob Error")
		return err
	}

	if err := ctrl.SetControllerReference(instance, job, r.Scheme); err != nil {
		log.V(1).Info("SetControllerReference Error")
		return err
	}

	// Check if the Job already exists
	foundJob := &unstructured.Unstructured{}
	foundJob.SetAPIVersion(kinds.DaskAPIVersion)
	foundJob.SetKind(kinds.DaskJobKind)
	justCreated := false
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundJob)
	if err != nil && apierrs.IsNotFound(err) {
		if instance.IsDone() {
			return nil
		}
		log.V(1).Info("Creating DaskJob", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Create(ctx, job)
		if err != nil {
			if updated := instance.LogWarning("OperatorCreateDaskJob", err.Error()); updated {
				log.V(1).Info("Warning unable to create DaskJob")
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
		log.V(1).Info("Updating DaskJob", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Update(ctx, foundJob)
		if err != nil {
			return err
		}
	}

	// Check the job status
	condUpdated, err := r.reconcileDaskJobStatus(instance, *foundJob)
	if err != nil {
		log.V(1).Info("reconcileDaskJobStatus Error")
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

func (r *OperationReconciler) reconcileDaskJobStatus(instance *operationv1.Operation, job unstructured.Unstructured) (bool, error) {
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
	jobStatus := daskapi.DaskJobStatus{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &jobStatus)
	if err != nil {
		log.Error(err, "Convert unstructured to status error")
		return false, err
	}

	if jobStatus.JobStatus == daskapi.DaskJobCreated {
		instance.LogWarning("Cluster created", "Waiting for scheduler and workers to start")
		log.V(1).Info("Job Logging Status Running")
		return true, nil
	}

	if jobStatus.JobStatus == daskapi.DaskJobRunning {
		instance.LogRunning()
		log.V(1).Info("Job Logging Status Running")
		return true, nil
	}

	if jobStatus.JobStatus == daskapi.DaskJobSuccessful {
		instance.LogSucceeded()
		instance.Status.CompletionTime = &now
		log.V(1).Info("Job Logging Status Succeeded")
		return true, nil
	}

	if jobStatus.JobStatus == daskapi.DaskJobFailed {
		newMessage := operationv1.GetFailureMessage("Job failed", podStatus, reason, message)
		if updated := instance.LogFailed(reason, newMessage); updated {
			instance.Status.CompletionTime = &now
			log.V(1).Info("Job Logging Status Failed", "Message", newMessage, "podStatus", podStatus, "PodMessage", message)
			return true, nil
		}
	}

	return false, nil
}

func (r *OperationReconciler) cleanUpDaskJob(ctx context.Context, instance *operationv1.Operation) (ctrl.Result, error) {
	return r.handleTTL(ctx, instance)
}
