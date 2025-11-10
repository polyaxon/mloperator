package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// Job is the Schema for the jobs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specifies the number of retries before marking this job failed.
	// +optional
	Termination TerminationSpec `json:"termination,omitempty" protobuf:"bytes,2,opt,name=termination"`

	// Flag to set a finalizer for collecting logs
	// +optional
	CollectLogs bool `json:"collectLogs" protobuf:"bytes,3,opt,name=collectLogs"`

	// Flag to set tell if Polyaxon should sync statuses with control plane
	// +optional
	SyncStatuses bool `json:"syncStatuses" protobuf:"bytes,4,opt,name=syncStatuses"`

	// Specification of the desired behavior of a job.
	// +optional
	BatchJobSpec *BatchJobSpec `json:"batchJobSpec,omitempty" protobuf:"bytes,6,opt,name=batchJobSpec"`

	// Current status of a job.
	// +optional
	Status OperationStatus `json:"status,omitempty" protobuf:"bytes,12,opt,name=status"`
}

// +kubebuilder:object:root=true

// JobList contains a list of Job
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Job{}, &JobList{})
}
