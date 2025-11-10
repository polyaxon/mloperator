package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// KfJob is the Schema for the kfjobs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KfJob struct {
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

	// Specification of the desired behavior of a TFJob.
	TFJobSpec *TFJobSpec `json:"tfJobSpec,omitempty" protobuf:"bytes,8,opt,name=tfJobSpec"`

	// Specification of the desired behavior of a PytorchJob.
	PytorchJobSpec *PytorchJobSpec `json:"pytorchJobSpec,omitempty" protobuf:"bytes,9,opt,name=pytorchJobSpec"`

	// Specification of the desired behavior of a MPIJob.
	MPIJobSpec *MPIJobSpec `json:"mpiJobSpec,omitempty" protobuf:"bytes,10,opt,name=mpiJobSpec"`

	// Current status of an op.
	// +optional
	Status OperationStatus `json:"status,omitempty" protobuf:"bytes,12,opt,name=status"`
}

// +kubebuilder:object:root=true

// KfJobList contains a list of KfJob
type KfJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KfJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KfJob{}, &KfJobList{})
}
