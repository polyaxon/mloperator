package kfjob

import (
	"context"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/kfjob/kinds"
	"github.com/polyaxon/mloperator/internal/helpers/managers"
)

func (r *KfJobReconciler) reconcileMPIJobOp(ctx context.Context, instance *apiv1.KfJob) (ctrl.Result, error) {
	// Reconcile the underlaying job
	return ctrl.Result{}, r.reconcileMPIJob(ctx, instance)
}

func (r *KfJobReconciler) reconcileMPIJob(ctx context.Context, instance *apiv1.KfJob) error {
	log := r.Log

	job, err := managers.GenerateMPIJob(
		instance.Name,
		instance.Namespace,
		instance.Labels,
		instance.Annotations,
		instance.Termination,
		*instance.MPIJobSpec,
	)
	if err != nil {
		log.V(1).Info("generateMPIJob Error")
		return err
	}

	if err := ctrl.SetControllerReference(instance, job, r.Scheme); err != nil {
		log.V(1).Info("SetControllerReference Error")
		return err
	}

	// Check if the Job already exists
	foundJob := &unstructured.Unstructured{}
	foundJob.SetAPIVersion(kinds.KFAPIVersion)
	foundJob.SetKind(kinds.MPIJobKind)
	justCreated := false
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundJob)
	if err != nil && apierrs.IsNotFound(err) {
		// Check if the job was previously created and should be marked as deleted
		shouldMarkDeleted, isDone := instance.Status.ShouldMarkJobAsDeleted()
		if isDone {
			return nil
		}

		if shouldMarkDeleted {
			log.Info("MPIJob was deleted externally, marking operation as stopped", "namespace", instance.Namespace, "name", instance.Name)
			if updated := instance.Status.MarkJobAsDeleted("MPIJob", false); updated {
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				_ = r.instanceSyncStatus(instance)
			}
			return nil
		}

		// Job never existed, create it
		log.V(1).Info("Creating MPIJob", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Create(ctx, job)
		if err != nil {
			if updated := instance.Status.LogWarning("OperatorCreateMPIJob", err.Error()); updated {
				log.V(1).Info("Warning unable to create MPIJob")
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
	} else if err != nil {
		return err
	}

	// Update the job object and write the result back if there are any changes
	// Don't update if: 1) just created, 2) instance is done, 3) MPIJob is in terminal state
	if !justCreated && !instance.Status.IsDone() && !isKfJobInTerminalState(foundJob) {
		if managers.CopyUnstructuredFields(job, foundJob) {
			log.V(1).Info("Updating MPIJob", "namespace", instance.Namespace, "name", instance.Name)
			err = r.Update(ctx, foundJob)
			if err != nil {
				return err
			}
		}
	}

	// Check the job status
	condUpdated, err := r.reconcileMPIJobStatus(instance, *foundJob)
	if err != nil {
		log.V(1).Info("reconcileMPIJobStatus Error")
		return err
	}
	if condUpdated {
		log.V(1).Info("Reconciling MPIJob status", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Status().Update(ctx, instance)
		if err != nil {
			return err
		}
		_ = r.instanceSyncStatus(instance)
	}

	return nil
}

func (r *KfJobReconciler) reconcileMPIJobStatus(instance *apiv1.KfJob, job unstructured.Unstructured) (bool, error) {
	return r.reconcileKfJobStatus(instance, job)
}

func (r *KfJobReconciler) cleanUpMPIJob(ctx context.Context, instance *apiv1.KfJob) (ctrl.Result, error) {
	return r.handleTTL(ctx, instance)
}
