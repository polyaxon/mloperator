package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// ServiceSpec defines the desired state of a service
// +k8s:openapi-gen=true
type ServiceSpec struct {
	// Replicas is the number of desired replicas.
	// This is a pointer to distinguish between explicit zero and unspecified.
	// Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty" default:"1" protobuf:"varint,1,opt,name=replicas"`

	// List of ports to expose on the service
	// +optional
	Ports []int32 `json:"ports,omitempty" protobuf:"varint,2,rep,name=ports"`

	// Is external service
	// +optional
	IsExternal bool `json:"isExternal,omitempty" protobuf:"varint,3,rep,name=IsExternal"`

	// Template describes the pods that will be created.
	Template corev1.PodTemplateSpec `json:"template" protobuf:"bytes,4,opt,name=template"`
}
