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
	ReplicaSpecs map[MXReplicaType]KFReplicaSpec `json:"replicaSpecs" protobuf:"bytes,4,opt,name=replicaSpecs"`
}

// MXReplicaType is the type for PyTorchReplica. Can be one of "Master" or "Worker".
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
