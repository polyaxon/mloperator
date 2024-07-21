package v1

// MXJobSpec defines the desired state of a mxnet job
// +k8s:openapi-gen=true
type MXJobSpec struct {
	// Defines the policy for cleaning up pods after the Job completes.
	// Defaults to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty" protobuf:"bytes,1,opt,name=cleanPodPolicy"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"  protobuf:"bytes,2,opt,name=schedulingPolicy"`

	// JobMode specify the kind of MXjob to do. Different mode may have
	// different MXReplicaSpecs request
	// optional
	JobMode MXJobModeType `json:"JobMode,omitempty"  protobuf:"bytes,3,opt,name=jobMode"`

	// A map of ReplicaType (type) to ReplicaSpec (value). Specifies the MXJob cluster configuration.
	// For example,
	//   {
	//     "Master": ReplicaSpec,
	//     "Worker": ReplicaSpec,
	//   }
	ReplicaSpecs map[MXReplicaType]*KFReplicaSpec `json:"replicaSpecs" protobuf:"bytes,4,opt,name=replicaSpecs"`
}

// MXReplicaType is the type for MXReplica. Can be one of "Master" or "Worker".
type MXReplicaType string

const (
	// MXReplicaTypeScheduler is the type of Master of distributed MXJjob
	MXReplicaTypeScheduler MXReplicaType = "Scheduler"

	// MXReplicaTypeServer is the type for workers of distributed MXJjob.
	MXReplicaTypeServer MXReplicaType = "Server"

	// MXReplicaTypeWorker is the type for workers of distributed MXJjob.
	MXReplicaTypeWorker MXReplicaType = "Worker"

	// MXReplicaTypeTunerTracker is the type for workers of distributed MXJjob.
	MXReplicaTypeTunerTracker MXReplicaType = "TunerTracker"

	// MXReplicaTypeTunerServer is the type for workers of distributed MXJjob.
	MXReplicaTypeTunerServer MXReplicaType = "TunerServer"

	// MXReplicaTypeTuner is the type for workers of distributed MXJjob.
	MXReplicaTypeTuner MXReplicaType = "Tuner"
)

// MXJobModeType id the type for JobMode
type MXJobModeType string

const (
	// Train Mode, in this mode requested MXReplicaSpecs need
	// has Server, Scheduler, Worker
	MXTrain MXJobModeType = "MXTrain"

	// Tune Mode, in this mode requested MXReplicaSpecs need
	// has Tuner
	MXTune MXJobModeType = "MXTune"
)
