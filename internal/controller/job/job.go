package job

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/helpers/managers"
)

func (r *JobReconciler) reconcileJobOp(ctx context.Context, instance *apiv1.Job) (ctrl.Result, error) {
	// Reconcile the underlaying job
	return ctrl.Result{}, r.reconcileJob(ctx, instance)
}

func (r *JobReconciler) reconcileJob(ctx context.Context, instance *apiv1.Job) error {
	log := r.Log

	job := managers.GenerateJob(
		instance.Name,
		instance.Namespace,
		instance.Labels,
		instance.Annotations,
		instance.Termination.BackoffLimit,
		instance.Termination.ActiveDeadlineSeconds,
		instance.Termination.TTLSecondsAfterFinished,
		instance.Termination.PodFailurePolicy,
		instance.BatchJobSpec.Template.Spec,
	)
	if err := ctrl.SetControllerReference(instance, job, r.Scheme); err != nil {
		log.V(1).Info("generateJob Error")
		return err
	}
	// Check if the Job already exists
	foundJob := &batchv1.Job{}
	justCreated := false
	err := r.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, foundJob)
	if err != nil && apierrs.IsNotFound(err) {
		// Check if the job was previously created and should be marked as deleted
		shouldMarkDeleted, isDone := instance.Status.ShouldMarkJobAsDeleted()
		if isDone {
			return nil
		}

		if shouldMarkDeleted {
			log.Info("Job was deleted externally, marking operation as failed", "namespace", job.Namespace, "name", job.Name)
			if updated := instance.Status.MarkJobAsDeleted("Kubernetes Job", false); updated {
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				_ = r.instanceSyncStatus(instance)
			}
			return nil
		}

		log.V(1).Info("Creating Job", "namespace", job.Namespace, "name", job.Name)
		err = r.Create(ctx, job)
		if err != nil {
			if updated := instance.Status.LogWarning("OperatorCreateJob", err.Error()); updated {
				log.Info("Warning unable to create Job")
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				_ = r.instanceSyncStatus(instance)
			}
			return err
		}
		justCreated = true
		instance.Status.LogStarting()
		err = r.Status().Update(ctx, instance)
		_ = r.instanceSyncStatus(instance)
	}
	if err != nil {
		return err
	}
	// Update the job object and write the result back if there are any changes
	if !justCreated && !instance.Status.IsDone() && !apiv1.IsOperationBeingDeleted(instance) && managers.CopyJobFields(job, foundJob) {
		log.V(1).Info("Updating Job", "namespace", job.Namespace, "name", job.Name)
		err = r.Update(ctx, foundJob)
		if err != nil {
			return err
		}
	}

	// Check the job status
	if condUpdated := r.reconcileJobStatus(instance, *foundJob); condUpdated {
		log.V(1).Info("Reconciling Job status", "namespace", job.Namespace, "name", job.Name)
		err = r.Status().Update(ctx, instance)
		if err != nil {
			return err
		}
		_ = r.instanceSyncStatus(instance)
	}

	return nil
}

func (r *JobReconciler) reconcileJobStatus(instance *apiv1.Job, job batchv1.Job) bool {
	now := metav1.Now()
	log := r.Log

	instanceID := instance.Labels["app.kubernetes.io/instance"]
	podStatus, reason, message := managers.HasUnschedulablePods(r.Client, instanceID, instance.Namespace)
	exitStatus, err := managers.GetMainContainerExitStatusByInstance(r.Client, instanceID, instance.Namespace)
	if err != nil {
		log.Error(err, "Get main container exit status error")
	}

	if len(job.Status.Conditions) > 0 {
		newJobCond := job.Status.Conditions[len(job.Status.Conditions)-1]

		if job.Status.Active == 0 && job.Status.Succeeded > 0 && managers.IsJobSucceeded(newJobCond) {
			if updated := instance.Status.LogSucceeded(); updated {
				instance.Status.CompletionTime = &now
				log.Info("Job Logging Status Succeeded")
				return true
			}
		}

		if job.Status.CompletionTime != nil && job.Status.Succeeded > 0 && managers.IsJobSucceeded(newJobCond) {
			if updated := instance.Status.LogSucceeded(); updated {
				instance.Status.CompletionTime = &now
				log.Info("Job Logging Status Succeeded with active non null")
				return true
			}
		}

		if job.Status.Failed > 0 && managers.IsJobFailed(newJobCond) {
			failedAttempts := job.Status.Failed
			var attemptPtr *int32
			if failedAttempts > 0 {
				attemptPtr = &failedAttempts
			}
			newMessage := managers.FormatMainContainerFailureMessage(newJobCond.Message, exitStatus, attemptPtr)
			if updated := instance.Status.LogFailed(newJobCond.Reason, newMessage); updated {
				instance.Status.CompletionTime = &now
				log.Info("Job failed", "Reason", newJobCond.Reason, "Message", newMessage, "Failed", job.Status.Failed, "podStatus", podStatus)
				return true
			}
		}
	}

	if podStatus == apiv1.OperationWarning {
		if updated := instance.Status.LogWarning(reason, message); updated {
			log.Info("Job has unschedulable pod(s)", "Reason", reason, "Message", message)
			return true
		}
		return false
	}

	if podStatus == apiv1.OperationStarting && job.Status.CompletionTime == nil {
		return false
	}

	if len(job.Status.Conditions) == 0 {
		if job.Status.Failed > 0 && job.Status.Active == 0 {
			if updated := instance.Status.LogWarning("", ""); updated {
				log.Info("Job has failed attempts", "Failed", job.Status.Failed, "Active", job.Status.Active)
				return true
			}
		} else if job.Status.Active > 0 {
			if updated := instance.Status.LogRunning(); updated {
				log.Info("Job is running", "Active", job.Status.Active)
				return true
			}
		}
		return false
	}
	return false
}

func (r *JobReconciler) cleanUpJob(ctx context.Context, instance *apiv1.Job) (ctrl.Result, error) {
	return r.handleTTL(ctx, instance)
}
