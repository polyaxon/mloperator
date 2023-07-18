package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// DaskReplicaSpec is a description of dask replica
// +k8s:openapi-gen=true
type DaskReplicaSpec struct {
	// Replicas is the desired number of replicas of the given template.
	// If unspecified, defaults to 1.
	Replicas int `json:"replicas,omitempty"`
	// Template is the object that describes the pod that
	// will be created for this replica. RestartPolicy in PodTemplateSpec
	// will be overide by RestartPolicy in ReplicaSpec
	Template corev1.PodTemplateSpec `json:"template,omitempty"`
	// Restart policy for all replicas within the job.
	// One of Always, OnFailure, Never and ExitCode.
	// Default to Never.
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`
}

// DaskJobSpec defines the desired state of a Dask job
// +k8s:openapi-gen=true
type DaskJobSpec struct {
	// Job replica spec
	Job DaskReplicaSpec `json:"head" protobuf:"bytes,3,opt,name=job"`
	// Worker replicas spec
	Worker DaskReplicaSpec `json:"workers" protobuf:"bytes,3,opt,name=worker"`
	// Scheduler replicas spec
	Scheduler DaskReplicaSpec `json:"scheduler" protobuf:"bytes,3,opt,name=scheduler"`
}
