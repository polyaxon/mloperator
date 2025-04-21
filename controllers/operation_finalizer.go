package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
)

// AddLogsFinalizer Adds finalizer by the reconciler
func (r *OperationReconciler) AddLogsFinalizer(ctx context.Context, instance *operationv1.Operation) error {
	instance.AddLogsFinalizer()
	return r.Update(ctx, instance)
}

// AddNotificationsFinalizer Adds finalizer by the reconciler
func (r *OperationReconciler) AddNotificationsFinalizer(ctx context.Context, instance *operationv1.Operation) error {
	instance.AddNotificationsFinalizer()
	return r.Update(ctx, instance)
}

func (r *OperationReconciler) handleFinalizers(ctx context.Context, instance *operationv1.Operation) error {
	log := r.Log

	if !instance.IsDone() {
		log.Info("Instance was probably stopped", "Append final status", "Stopping")
		r.syncStatus(
			instance,
			operationv1.NewOperationCondition(
				operationv1.OperationStopped,
				corev1.ConditionTrue,
				"OperatorFinalizer",
				"Operation stopped in finalizer",
			),
		)
	}

	if instance.HasLogsFinalizer() {
		if err := r.collectLogs(instance); err != nil {
			log.Info("Error logs collection", "Error", err.Error())
		}

		instance.RemoveLogsFinalizer()
		return r.Update(ctx, instance)
	}

	if instance.HasNotificationsFinalizer() {
		if err := r.notify(instance); err != nil {
			log.Info("Error notification", "Error", err.Error())
		}

		instance.RemoveNotificationsFinalizer()
		return r.Update(ctx, instance)
	}

	return nil
}
