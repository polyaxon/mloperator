package job

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
)

// AddLogsFinalizer Adds finalizer by the reconciler
func (r *JobReconciler) AddLogsFinalizer(ctx context.Context, instance *apiv1.Job) error {
	controllerutil.AddFinalizer(instance, apiv1.OperationLogsFinalizer)
	return r.Update(ctx, instance)
}

// AddStatusFinalizer Adds finalizer by the reconciler
func (r *JobReconciler) AddStatusFinalizer(ctx context.Context, instance *apiv1.Job) error {
	controllerutil.AddFinalizer(instance, apiv1.OperationStatusFinalizer)
	return r.Update(ctx, instance)
}

func (r *JobReconciler) handleFinalizers(ctx context.Context, instance *apiv1.Job) error {
	log := r.Log

	if !instance.Status.IsDone() {
		log.Info("Instance was probably stopped", "Append final status", "Stopping")
		r.syncStatus(
			instance,
			apiv1.NewOperationCondition(
				apiv1.OperationStopped,
				corev1.ConditionTrue,
				"OperatorFinalizer",
				"Operation stopped in finalizer",
			),
		)
	}

	if controllerutil.ContainsFinalizer(instance, apiv1.OperationLogsFinalizer) {
		if err := r.collectLogs(instance); err != nil {
			log.Info("Error logs collection", "Error", err.Error())
			// TODO: add better error handling
			return nil
		}

		controllerutil.RemoveFinalizer(instance, apiv1.OperationLogsFinalizer)
		return r.Update(ctx, instance)
	}

	if controllerutil.ContainsFinalizer(instance, apiv1.OperationStatusFinalizer) {
		controllerutil.RemoveFinalizer(instance, apiv1.OperationStatusFinalizer)
		return r.Update(ctx, instance)
	}

	return nil
}
