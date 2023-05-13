package v1

// OperationLogsFinalizer registration
const OperationLogsFinalizer = "operation.logs.finalizers.polyaxon.com"

// HasLogsFinalizer check for Operation
func (instance *Operation) HasLogsFinalizer() bool {
	return containsString(instance.ObjectMeta.Finalizers, OperationLogsFinalizer)
}

// AddLogsFinalizer handler for Operation
func (instance *Operation) AddLogsFinalizer() {
	instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, OperationLogsFinalizer)
}

// RemoveLogsFinalizer handler for Operation
func (instance *Operation) RemoveLogsFinalizer() {
	instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, OperationLogsFinalizer)
}

// OperationNotificationsFinalizer registration
const OperationNotificationsFinalizer = "operation.notifications.finalizers.polyaxon.com"

// HasNotificationsFinalizer check for Operation
func (instance *Operation) HasNotificationsFinalizer() bool {
	return containsString(instance.ObjectMeta.Finalizers, OperationNotificationsFinalizer)
}

// AddNotificationsFinalizer handler for Operation
func (instance *Operation) AddNotificationsFinalizer() {
	instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, OperationNotificationsFinalizer)
}

// RemoveNotificationsFinalizer handler for Operation
func (instance *Operation) RemoveNotificationsFinalizer() {
	instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, OperationNotificationsFinalizer)
}
