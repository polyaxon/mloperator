/*
Copyright 2018-2021 Polyaxon, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

	// A map of ReplicaType (type) to ReplicaSpec (value). Specifies the PyTorch cluster configuration.
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
