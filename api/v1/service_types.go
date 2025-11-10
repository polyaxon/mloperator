package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// Service is the Schema for the services API
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName=svc
// +kubebuilder:subresource:status
type Service struct {
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

	// Specification of the desired behavior of a Service.
	ServiceSpec *ServiceSpec `json:"serviceSpec,omitempty" protobuf:"bytes,7,opt,name=serviceSpec"`

	// Current status of an op.
	// +optional
	Status OperationStatus `json:"status,omitempty" protobuf:"bytes,8,opt,name=status"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
