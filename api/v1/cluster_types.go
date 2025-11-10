package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// Cluster is the Schema for the clusters API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Cluster struct {
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

	// Specification of the desired behavior of a DaskCluster.
	DaskClusterSpec *DaskClusterSpec `json:"daskClusterSpec,omitempty" protobuf:"bytes,14,opt,name=daskClusterSpec"`

	// Specification of the desired behavior of a RayCluster.
	RayClusterSpec *RayClusterSpec `json:"rayClusterSpec,omitempty" protobuf:"bytes,15,opt,name=rayClusterSpec"`

	// Current status of an op.
	// +optional
	Status OperationStatus `json:"status,omitempty" protobuf:"bytes,11,opt,name=status"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
