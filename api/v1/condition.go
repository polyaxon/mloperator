package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OperationCondition defines the conditions of Operation or OpService
// +k8s:openapi-gen=true
type OperationCondition struct {
	// Type is the type of the condition.
	Type OperationConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`

	// The last time this condition was updated.
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,3,opt,name=lastUpdateTime"`

	// Last time the condition transitioned.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`

	// The reason for this container condition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// OperationTriggerNotificationType maps the notifiable conditions
// +k8s:openapi-gen=true
type OperationTriggerNotificationType string

const (
	// OperationSucceededTrigger means underlaying Operation has completed successfully.
	OperationSucceededTrigger OperationTriggerNotificationType = "Succeeded"
	// OperationFailedTrigger means underlaying Operation has failed.
	OperationFailedTrigger OperationTriggerNotificationType = "Failed"
	// OperationStoppedTrigger means that the Operation was stopped/killed.
	OperationStoppedTrigger OperationTriggerNotificationType = "Stopped"
	// OperationDoneTrigger means that the Operation was stopped/killed.
	OperationDoneTrigger OperationTriggerNotificationType = "Done"
)

// OperationConditionType maps the conditions of a job or service once deployed
// +k8s:openapi-gen=true
type OperationConditionType string

const (
	// OperationStarting means underlaying Operation has started.
	OperationStarting OperationConditionType = "Starting"
	// OperationRunning means underlaying Operation is running,
	OperationRunning OperationConditionType = "Running"
	// OperationWarning means underlaying Operation has some issues.
	OperationWarning OperationConditionType = "Warning"
	// OperationSucceeded means underlaying Operation has completed successfully.
	OperationSucceeded OperationConditionType = "Succeeded"
	// OperationFailed means underlaying Operation has failed.
	OperationFailed OperationConditionType = "Failed"
	// OperationStopped means that the Operation was stopped/killed.
	OperationStopped OperationConditionType = "Stopped"
)

// NewOperationCondition makes a new instance of OperationCondition
func NewOperationCondition(conditionType OperationConditionType, status corev1.ConditionStatus, reason, message string) OperationCondition {
	return OperationCondition{
		Type:               conditionType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func GetMessage(condition OperationConditionType, entityMessage string, status OperationConditionType, reason string, message string) string {
	newMessage := entityMessage
	if status == condition && message != "" {
		newMessage = newMessage + " (Pod: <reason: " + reason + ", message " + message + ")"
	}
	return newMessage
}

func GetFailureMessage(entityMessage string, status OperationConditionType, reason string, message string) string {
	return GetMessage(OperationFailed, entityMessage, status, reason, message)
}

func GetStoppedMessage(entityMessage string, status OperationConditionType, reason string, message string) string {
	return GetMessage(OperationStopped, entityMessage, status, reason, message)
}

// getOrUpdateOperationCondition get new or updated version of current confition or returns nil if nothing changed
func getOrUpdateOperationCondition(currentCond *OperationCondition, conditionType OperationConditionType, status corev1.ConditionStatus, reason, message string) (*OperationCondition, bool) {
	newCond := NewOperationCondition(conditionType, status, reason, message)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Type == newCond.Type && currentCond.Status == newCond.Status && currentCond.Reason == newCond.Reason {
		// Always update final states
		if currentCond.Type == OperationSucceeded || currentCond.Type == OperationFailed || currentCond.Type == OperationStopped {
			return &newCond, true
		}
		return &newCond, false
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == newCond.Status {
		newCond.LastTransitionTime = currentCond.LastTransitionTime
	}

	return &newCond, true
}

// getLastEntityCondition returns the condition with the specific type form status.conditions
func getLastEntityCondition(status OperationStatus, condType OperationConditionType) *OperationCondition {
	if len(status.Conditions) > 0 {
		return &status.Conditions[len(status.Conditions)-1]
	}
	return nil
}

// getEntityConditionFromStatus returns the condition with the specific type form status.conditions
func getEntityConditionFromStatus(status OperationStatus, condType OperationConditionType) *OperationCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

// hasOperationCondition checks if a status has a specific condition type
func hasOperationCondition(status OperationStatus, condType OperationConditionType) bool {
	cond := getEntityConditionFromStatus(status, condType)
	if cond != nil && cond.Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// hasOperationCondition checks if a status's last codition is of a specific type
func hasLastOperationCondition(status OperationStatus, condType OperationConditionType) bool {
	cond := status.Conditions[len(status.Conditions)-1]
	if cond.Type == condType && cond.Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsStarting checks if an ml operation status is in starting condition
func (status *OperationStatus) IsStarting() bool {
	return hasLastOperationCondition(*status, OperationStarting)
}

// IsRunning checks if an ml operation status is in running condition
func (status *OperationStatus) IsRunning() bool {
	return hasLastOperationCondition(*status, OperationRunning)
}

// IsWarning checks if an ml operation status is in warning condition
func (status *OperationStatus) IsWarning() bool {
	return hasLastOperationCondition(*status, OperationWarning)
}

// IsSucceeded checks if an ml operation status is succeeded
func (status *OperationStatus) IsSucceeded() bool {
	return hasOperationCondition(*status, OperationSucceeded)
}

// IsFailed checks if an ml operation status is failed
func (status *OperationStatus) IsFailed() bool {
	return hasOperationCondition(*status, OperationFailed)
}

// IsStopped checks if an ml operation status is stopped
func (status *OperationStatus) IsStopped() bool {
	return hasOperationCondition(*status, OperationStopped)
}

// IsDone checks if it the Operation reached a final condition
func (status *OperationStatus) IsDone() bool {
	return status.IsSucceeded() || status.IsFailed() || status.IsStopped()
}

// IsOperationBeingDeleted checks if a Kubernetes resource is being deleted
// This works with any type that implements metav1.Object (Job, Service, Operation, etc.)
func IsOperationBeingDeleted[T metav1.Object](obj T) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

// removeCondition removes a condition of the specified type from the status
func removeCondition(status *OperationStatus, conditionType OperationConditionType) {
	var newConditions []OperationCondition
	for _, c := range status.Conditions {
		if c.Type == conditionType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	status.Conditions = newConditions
}

// logCondition logs a condition to the status
func logCondition(status *OperationStatus, condType OperationConditionType, conditionStatus corev1.ConditionStatus, reason, message string) bool {
	currentCond := getLastEntityCondition(*status, condType)
	cond, isUpdated := getOrUpdateOperationCondition(currentCond, condType, conditionStatus, reason, message)
	if isUpdated && cond != nil {
		removeCondition(status, condType)
		status.Conditions = append(status.Conditions, *cond)
		return true
	}
	return false
}

// LogStarting sets Operation to starting
func (status *OperationStatus) LogStarting() bool {
	return logCondition(status, OperationStarting, corev1.ConditionTrue, "OperatorController", "Operation is starting")
}

// LogRunning sets Operation to running
func (status *OperationStatus) LogRunning() bool {
	return logCondition(status, OperationRunning, corev1.ConditionTrue, "OperatorController", "Operation is running")
}

// LogWarning sets Operation to succeeded
func (status *OperationStatus) LogWarning(reason, message string) bool {
	if reason == "" {
		reason = "OperatorController"
	}
	if message == "" {
		message = "Underlaying job has an issue"
	}
	return logCondition(status, OperationWarning, corev1.ConditionTrue, reason, message)
}

// LogSucceeded sets Operation to succeeded
func (status *OperationStatus) LogSucceeded() bool {
	return logCondition(status, OperationSucceeded, corev1.ConditionTrue, "OperatorController", "Operation has succeeded")
}

// LogFailed sets Operation to failed
func (status *OperationStatus) LogFailed(reason, message string) bool {
	return logCondition(status, OperationFailed, corev1.ConditionTrue, reason, message)
}

// LogStopped sets Operation to stopped
func (status *OperationStatus) LogStopped(reason, message string) bool {
	return logCondition(status, OperationStopped, corev1.ConditionTrue, reason, message)
}

// ShouldMarkJobAsDeleted checks if a job that doesn't exist should be marked as deleted
// rather than being recreated. It returns true if the job was previously created and running
// but is now missing (deleted externally).
//
// This function checks if the operation has progressed beyond initial creation by examining
// the status conditions. If the last condition indicates the job was running or had warnings,
// it means the job existed before and should not be recreated.
//
// Returns:
//   - shouldMarkDeleted: true if the job should be marked as deleted instead of recreated
//   - isDone: true if the operation is already in a terminal state
func (status *OperationStatus) ShouldMarkJobAsDeleted() (shouldMarkDeleted bool, isDone bool) {
	// If operation is already done, don't recreate or mark as deleted
	if status.IsDone() {
		return false, true
	}

	// Check if the job was previously created (status has progressed beyond creation)
	if len(status.Conditions) == 0 {
		// No conditions mean the job was never created, so it's safe to create it
		return false, false
	}

	lastCond := status.Conditions[len(status.Conditions)-1]

	// If the operation was previously running or had warnings, it means
	// the job existed before and was deleted externally
	if lastCond.Type == OperationRunning || lastCond.Type == OperationWarning {
		return true, false
	}

	// For other states (Starting, etc.), allow job creation
	return false, false
}

// MarkJobAsDeleted marks an operation as failed or stopped due to external job deletion.
// This is a helper function that sets the appropriate status when a job is detected
// as deleted externally.
//
// Parameters:
//   - jobType: a descriptive name for the job type (e.g., "Kubernetes Job", "TFJob")
//   - useFailed: if true, marks as Failed; if false, marks as Stopped
//
// Returns true if the status was updated
func (status *OperationStatus) MarkJobAsDeleted(jobType string, useFailed bool) bool {
	now := metav1.Now()
	message := "The underlying " + jobType + " was deleted externally"

	var updated bool
	if useFailed {
		updated = status.LogFailed("JobDeleted", message)
	} else {
		updated = status.LogStopped("JobDeleted", message)
	}

	if updated {
		status.CompletionTime = &now
	}

	return updated
}
