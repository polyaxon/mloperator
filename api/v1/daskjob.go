package v1

// DaskJobSpec defines the desired state of a Dask job
// +k8s:openapi-gen=true
type DaskJobSpec struct {
	// Defines the policy for cleaning up pods after the Job completes.
	// Defaults to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty" protobuf:"bytes,1,opt,name=cleanPodPolicy"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"  protobuf:"bytes,2,opt,name=schedulingPolicy"`

	// A map of ReplicaType (type) to ReplicaSpec (value). Specifies the Dask cluster configuration.
	// For example,
	//   {
	//     "Master": DaskReplicaSpec,
	//     "Worker": DaskReplicaSpec,
	//   }
	ReplicaSpecs map[DaskReplicaType]KFReplicaSpec `json:"replicaSpecs" protobuf:"bytes,3,opt,name=replicaSpecs"`
}

// DaskReplicaType is the type for DaskReplica. Can be one of "Master" or "Worker".
type DaskReplicaType string

const (
	// DaskReplicaTypeMaster is the type of Master of distributed Dask
	DaskReplicaTypeMaster DaskReplicaType = "Master"

	// DaskReplicaTypeWorker is the type for workers of distributed Dask.
	DaskReplicaTypeWorker DaskReplicaType = "Worker"
)
