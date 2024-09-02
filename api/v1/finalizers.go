package v1

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// OperationLogsFinalizer registration
const OperationLogsFinalizer = "operation.logs.finalizers.polyaxon.com"

// HasLogsFinalizer check for Operation
func (instance *Operation) HasLogsFinalizer() bool {
	return controllerutil.ContainsFinalizer(instance, OperationLogsFinalizer)
}

// AddLogsFinalizer handler for Operation
func (instance *Operation) AddLogsFinalizer() {
	controllerutil.AddFinalizer(instance, OperationLogsFinalizer)
}

// RemoveLogsFinalizer handler for Operation
func (instance *Operation) RemoveLogsFinalizer() {
	controllerutil.RemoveFinalizer(instance, OperationLogsFinalizer)
}

// OperationNotificationsFinalizer registration
const OperationNotificationsFinalizer = "operation.notifications.finalizers.polyaxon.com"

// HasNotificationsFinalizer check for Operation
func (instance *Operation) HasNotificationsFinalizer() bool {
	return controllerutil.ContainsFinalizer(instance, OperationNotificationsFinalizer)
}

// AddNotificationsFinalizer handler for Operation
func (instance *Operation) AddNotificationsFinalizer() {
	controllerutil.AddFinalizer(instance, OperationNotificationsFinalizer)
}

// RemoveNotificationsFinalizer handler for Operation
func (instance *Operation) RemoveNotificationsFinalizer() {
	controllerutil.RemoveFinalizer(instance, OperationNotificationsFinalizer)
}
