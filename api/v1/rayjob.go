package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// KFReplicaSpec is a description of kubeflow replica
// +k8s:openapi-gen=true
type RayReplicaSpec struct {
	// we can have multiple worker groups, we distinguish them by name
	GroupName string `json:"groupName"`
	// Replicas is the desired number of replicas of the given template.
	// If unspecified, defaults to 1.
	Replicas *int32 `json:"replicas,omitempty"`
	// MinReplicas defaults to 1
	MinReplicas *int32 `json:"minReplicas"`
	// MaxReplicas defaults to maxInt32
	MaxReplicas *int32 `json:"maxReplicas"`
	// RayStartParams are the params of the start command: address, object-store-memory, ...
	RayStartParams map[string]string `json:"rayStartParams"`
	// Template is the object that describes the pod that
	// will be created for this replica. RestartPolicy in PodTemplateSpec
	// will be overide by RestartPolicy in ReplicaSpec
	Template corev1.PodTemplateSpec `json:"template,omitempty"`
	// Restart policy for all replicas within the job.
	// One of Always, OnFailure, Never and ExitCode.
	// Default to Never.
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`
}

// RayJobSpec defines the desired state of a Ray job
// +k8s:openapi-gen=true
type RayJobSpec struct {
	// Defines the policy for cleaning up pods after the Job completes.
	// Defaults to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty" protobuf:"bytes,1,opt,name=cleanPodPolicy"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"  protobuf:"bytes,2,opt,name=schedulingPolicy"`

	// Head replica spec
	Head KFReplicaSpec `json:"head" protobuf:"bytes,3,opt,name=head"`
	// Worker replicas spec
	Workers []KFReplicaSpec `json:"workers" protobuf:"bytes,3,opt,name=workers"`
}
