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
	// A map of ReplicaType (type) to ReplicaSpec (value). Specifies the Dask cluster configuration.
	// For example,
	//   {
	//     "Job": DaskReplicaSpec,
	//     "Worker": DaskReplicaSpec,
	//     "Scheduler": DaskReplicaSpec,
	//   }
	ReplicaSpecs map[DaskReplicaType]DaskReplicaSpec `json:"replicaSpecs" protobuf:"bytes,4,opt,name=replicaSpecs"`

	Service corev1.ServiceSpec `json:"service" protobuf:"bytes,5,opt,name=service"`
}

// DaskReplicaType is the type for DaskReplica. Can be one of "Job" or "Worker" or "Scheduler".
type DaskReplicaType string

const (
	// DaskReplicaTypeJob is the type of Master of distributed Dask
	DaskReplicaTypeJob DaskReplicaType = "Job"

	// DaskReplicaTypeWorker is the type for workers of distributed Dask.
	DaskReplicaTypeWorker DaskReplicaType = "Worker"

	// DaskReplicaTypeScheduler is the type for workers of distributed Dask.
	DaskReplicaTypeScheduler DaskReplicaType = "Scheduler"
)
