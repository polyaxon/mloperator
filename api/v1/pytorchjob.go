package v1

// PytorchJobSpec defines the desired state of a pytorch job
// +k8s:openapi-gen=true
type PytorchJobSpec struct {
	// Defines the policy for cleaning up pods after the Job completes.
	// Defaults to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty" protobuf:"bytes,1,opt,name=cleanPodPolicy"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"  protobuf:"bytes,2,opt,name=schedulingPolicy"`

	// A map of ReplicaType (type) to ReplicaSpec (value). Specifies the PyTorch cluster configuration.
	// For example,
	//   {
	//     "Master": PyTorchReplicaSpec,
	//     "Worker": PyTorchReplicaSpec,
	//   }
	ReplicaSpecs map[PyTorchReplicaType]KFReplicaSpec `json:"replicaSpecs" protobuf:"bytes,3,opt,name=replicaSpecs"`
}

// PyTorchReplicaType is the type for PyTorchReplica. Can be one of "Master" or "Worker".
type PyTorchReplicaType string

const (
	// PyTorchReplicaTypeMaster is the type of Master of distributed PyTorch
	PyTorchReplicaTypeMaster PyTorchReplicaType = "Master"

	// PyTorchReplicaTypeWorker is the type for workers of distributed PyTorch.
	PyTorchReplicaTypeWorker PyTorchReplicaType = "Worker"
)
