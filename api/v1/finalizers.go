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

// OperationStatusFinalizer registration (this holds the operation until it is fully deleted)
const OperationStatusFinalizer = "operation.status.finalizers.polyaxon.com"

// HasStatusFinalizer check for Operation
func (instance *Operation) HasStatusFinalizer() bool {
	return controllerutil.ContainsFinalizer(instance, OperationStatusFinalizer)
}

// AddStatusFinalizer handler for Operation
func (instance *Operation) AddStatusFinalizer() {
	controllerutil.AddFinalizer(instance, OperationStatusFinalizer)
}

// RemoveStatusFinalizer handler for Operation
func (instance *Operation) RemoveStatusFinalizer() {
	controllerutil.RemoveFinalizer(instance, OperationStatusFinalizer)
}
