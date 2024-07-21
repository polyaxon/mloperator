package v1

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

// PaddleJobSpec defines the desired state of a paddle job
// +k8s:openapi-gen=true
type PaddleJobSpec struct {
	// Defines the policy for cleaning up pods after the Job completes.
	// Defaults to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty" protobuf:"bytes,1,opt,name=cleanPodPolicy"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"  protobuf:"bytes,2,opt,name=schedulingPolicy"`

	// ElasticPolicy holds the elastic policy for paddle job.
	ElasticPolicy *PaddleElasticPolicy `json:"elasticPolicy,omitempty"  protobuf:"bytes,3,opt,name=elasticPolicy"`

	// A map of ReplicaType (type) to ReplicaSpec (value). Specifies the Paddle cluster configuration.
	// For example,
	//   {
	//     "Master": ReplicaSpec,
	//     "Worker": ReplicaSpec,
	//   }
	ReplicaSpecs map[PaddleReplicaType]*KFReplicaSpec `json:"replicaSpecs" protobuf:"bytes,4,opt,name=replicaSpecs"`
}

type PaddleElasticPolicy struct {
	// minReplicas is the lower limit for the number of replicas to which the training job
	// can scale down.  It defaults to null.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas, defaults to null.
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// MaxRestarts is the limit for restart times of pods in elastic mode.
	// +optional
	MaxRestarts *int32 `json:"maxRestarts,omitempty"`

	// Metrics contains the specifications which are used to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used).  The desired replica count is calculated with multiplying the
	// ratio between the target value and the current value by the current
	// number of pods. Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa.  See the individual metric source types for
	// more information about how each type of metric must respond.
	// If not set, the HPA will not be created.
	// +optional
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`
}

// TFReplicaType is the type for TFReplica. Can be one of: "Chief"/"Master" (semantically equivalent),
// "Worker", "PS", or "Evaluator".
type PaddleReplicaType string

const (
	// PaddleoostReplicaTypeMaster is the type of Master of distributed PaddleoostJjob
	PaddleoostReplicaTypeMaster PaddleReplicaType = "Master"

	// PaddleoostReplicaTypeWorker is the type for workers of distributed PaddleoostJjob.
	PaddleoostReplicaTypeWorker PaddleReplicaType = "Worker"
)
