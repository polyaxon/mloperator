package v1

// MPIJobSpec defines the desired state of an mpi job
// +k8s:openapi-gen=true
type MPIJobSpec struct {

	// Defines the policy for cleaning up pods after the Job completes.
	// Defaults to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty" protobuf:"bytes,1,opt,name=cleanPodPolicy"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"  protobuf:"bytes,2,opt,name=schedulingPolicy"`

	// CleanPodPolicy defines the policy that whether to kill pods after the job completes.
	// Defaults to None.
	SlotsPerWorker *int32 `json:"slotsPerWorker,omitempty" protobuf:"bytes,3,opt,name=slotsPerWorker"`

	// `MPIReplicaSpecs` contains maps from `MPIReplicaType` to `ReplicaSpec` that
	// specify the MPI replicas to run.
	ReplicaSpecs map[MPIReplicaType]*KFReplicaSpec `json:"replicaSpecs" protobuf:"bytes,6,opt,name=replicaSpecs"`
}

// MPIReplicaType is the type for MPIReplica.
type MPIReplicaType string

const (
	// MPIReplicaTypeLauncher is the type for launcher replica.
	MPIReplicaTypeLauncher MPIReplicaType = "Launcher"

	// MPIReplicaTypeWorker is the type for worker replicas.
	MPIReplicaTypeWorker MPIReplicaType = "Worker"
)
