package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OperationStatus defines the observed state of a job or a service
// +k8s:openapi-gen=true
type OperationStatus struct {
	// The latest available observations of an object's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []OperationCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Represents the time when the job was acknowledged by the controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty" protobuf:"bytes,2,opt,name=startTime"`

	// Represents the time when the job was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty" protobuf:"bytes,3,opt,name=startTime"`

	// Represents the last time when the job was reconciled.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty" protobuf:"bytes,4,opt,name=startTime"`
}
