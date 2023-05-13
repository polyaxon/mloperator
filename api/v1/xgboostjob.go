package v1

// XGBoostJobSpec defines the desired state of a xgboost job
// +k8s:openapi-gen=true
type XGBoostJobSpec struct {
	// Defines the policy for cleaning up pods after the Job completes.
	// Defaults to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty" protobuf:"bytes,1,opt,name=cleanPodPolicy"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"  protobuf:"bytes,2,opt,name=schedulingPolicy"`

	// A map of ReplicaType (type) to ReplicaSpec (value). Specifies the XGBoost cluster configuration.
	// For example,
	//   {
	//     "Master": ReplicaSpec,
	//     "Worker": ReplicaSpec,
	//   }
	ReplicaSpecs map[XGBReplicaType]KFReplicaSpec `json:"replicaSpecs" protobuf:"bytes,3,opt,name=replicaSpecs"`
}

// TFReplicaType is the type for TFReplica. Can be one of: "Chief"/"Master" (semantically equivalent),
// "Worker", "PS", or "Evaluator".
type XGBReplicaType string

const (
	// XGBoostReplicaTypeMaster is the type of Master of distributed XGBoostJjob
	XGBoostReplicaTypeMaster XGBReplicaType = "Scheduler"

	// XGBoostReplicaTypeWorker is the type for workers of distributed XGBoostJjob.
	XGBoostReplicaTypeWorker XGBReplicaType = "Worker"
)
