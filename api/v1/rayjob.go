package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// RayReplicaSpec is a description of ray replica
// +k8s:openapi-gen=true
type RayReplicaSpec struct {
	// we can have multiple worker groups, we distinguish them by name
	GroupName string `json:"groupName,omitempty"`
	// Replicas is the desired number of replicas of the given template.
	// If unspecified, defaults to 1.
	Replicas *int32 `json:"replicas,omitempty"`
	// MinReplicas defaults to 1
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// MaxReplicas defaults to maxInt32
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	// RayStartParams are the params of the start command: address, object-store-memory, ...
	RayStartParams map[string]string `json:"rayStartParams,omitempty"`
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
	// Important: Run "make" to regenerate code after modifying this file
	Entrypoint string `json:"entrypoint,omitempty"`
	// Metadata is data to store along with this job.
	Metadata map[string]string `json:"metadata,omitempty"`
	// RuntimeEnv yaml representing the runtime environment
	RuntimeEnv string `json:"runtimeEnv,omitempty"`
	// RayVersion is the version of ray being used. This determines the autoscaler's image version.
	RayVersion string `json:"rayVersion,omitempty"`
	// Head replica spec
	Head RayReplicaSpec `json:"head" protobuf:"bytes,3,opt,name=head"`
	// Worker replicas spec
	Workers []RayReplicaSpec `json:"workers" protobuf:"bytes,3,opt,name=workers"`
}
