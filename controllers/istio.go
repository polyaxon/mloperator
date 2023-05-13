package controllers

import (
	"context"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/kinds"
	"github.com/polyaxon/mloperator/controllers/managers"
)

func (r *OperationReconciler) reconcileVirtualService(ctx context.Context, instance *operationv1.Operation) error {
	log := r.Log

	virtualservice, err := managers.GenerateVirtualService(instance.Name, instance.Namespace)
	if err != nil {
		log.V(1).Info("generateVirtualService Error")
		return err
	}
	if err := ctrl.SetControllerReference(instance, virtualservice, r.Scheme); err != nil {
		log.V(1).Info("SetControllerReference Error")
		return err
	}

	// Check if the Service already exists
	foundVirtualService := &unstructured.Unstructured{}
	foundVirtualService.SetAPIVersion(kinds.IstioAPIVersion)
	foundVirtualService.SetKind(kinds.IstioVirtualServiceKind)
	justCreated := false
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundVirtualService)
	if err != nil && apierrs.IsNotFound(err) {
		if instance.IsDone() {
			return nil
		}
		log.V(1).Info("Creating Virtual Service", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Create(ctx, virtualservice)
		if err != nil {
			if updated := instance.LogWarning("OperatorCreateVirtualService", err.Error()); updated {
				log.V(1).Info("Warning unable to create VirtualService")
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				r.instanceSyncStatus(instance)
			}
			return err
		}
		justCreated = true
	} else if err != nil {
		return err
	}

	// Update the servuce object and write the result back if there are any changes
	if !justCreated && managers.CopyVirtualService(virtualservice, foundVirtualService) {
		log.V(1).Info("Updating virtual service\n", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Update(ctx, foundVirtualService)
		if err != nil {
			return err
		}
	}

	return nil
}
