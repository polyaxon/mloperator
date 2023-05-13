package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// TODO: integrate this it when https://github.com/kubernetes/kubernetes/issues/28486 has been fixed
// Optional number of failed pods to retain. This will be especially good for when restart is True since the underlaying pods will disapear.

// BatchJobSpec defines the desired state of a batch job
// +k8s:openapi-gen=true
type BatchJobSpec struct {
	// Template describes the pods that will be created.
	Template corev1.PodTemplateSpec `json:"template" protobuf:"bytes,1,opt,name=template"`
}
