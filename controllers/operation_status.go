package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
)

// AddStartTime Adds starttime field by the reconciler
func (r *OperationReconciler) AddStartTime(ctx context.Context, instance *operationv1.Operation) error {
	if instance.Status.StartTime != nil {
		return nil
	}

	now := metav1.Now()
	log := r.Log

	log.V(1).Info("Setting StartTime", "Operation", instance.Name)
	instance.Status.StartTime = &now
	return r.Update(ctx, instance)
}
