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

package kfapi

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
)

// JobStatus represents the current observed state of the training Job.
type JobStatus struct {
	// Conditions is an array of current observed job conditions.
	Conditions []JobCondition `json:"conditions"`

	// ReplicaStatuses is map of ReplicaType and ReplicaStatus,
	// specifies the status of each replica.
	ReplicaStatuses map[ReplicaType]*ReplicaStatus `json:"replicaStatuses"`

	// Represents time when the job was acknowledged by the job controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the job was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Represents last time when the job was reconciled. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// ReplicaType represents the type of the replica. Each operator needs to define its
// own set of ReplicaTypes.
type ReplicaType string

// JobCondition describes the state of the job at a certain point.
type JobCondition struct {
	// Type of job condition.
	Type JobConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// JobConditionType defines all kinds of types of JobStatus.
type JobConditionType string

const (
	// JobCreated means the job has been accepted by the system,
	// but one or more of the pods/services has not been started.
	// This includes time before pods being scheduled and launched.
	JobCreated JobConditionType = "Created"

	// JobRunning means all sub-resources (e.g. services/pods) of this job
	// have been successfully scheduled and launched.
	// The training is running without error.
	JobRunning JobConditionType = "Running"

	// JobRestarting means one or more sub-resources (e.g. services/pods) of this job
	// reached phase failed but maybe restarted according to it's restart policy
	// which specified by user in v1.PodTemplateSpec.
	// The training is freezing/pending.
	JobRestarting JobConditionType = "Restarting"

	// JobSucceeded means all sub-resources (e.g. services/pods) of this job
	// reached phase have terminated in success.
	// The training is complete without error.
	JobSucceeded JobConditionType = "Succeeded"

	// JobFailed means one or more sub-resources (e.g. services/pods) of this job
	// reached phase failed with no restarting.
	// The training has failed its execution.
	JobFailed JobConditionType = "Failed"
)

// ReplicaStatus represents the current observed state of the replica.
type ReplicaStatus struct {
	// The number of actively running pods.
	Active int32 `json:"active,omitempty"`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed,omitempty"`
}

type RunPolicy struct {
	// CleanPodPolicy defines the policy to kill pods after the job completes.
	// Default to Running.
	CleanPodPolicy *operationv1.CleanPodPolicy `json:"cleanPodPolicy,omitempty"`

	// TTLSecondsAfterFinished is the TTL to clean up jobs.
	// It may take extra ReconcilePeriod seconds for the cleanup, since
	// reconcile gets called periodically.
	// Default to infinite.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// Specifies the duration in seconds relative to the startTime that the job may be active
	// before the system tries to terminate it; value must be positive integer.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// Optional number of retries before marking this job failed.
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *operationv1.SchedulingPolicy `json:"schedulingPolicy,omitempty"`
}

// MPIJobSpec resource definiton.
type MPIJobSpec struct {
	SlotsPerWorker    *int32                                                    `json:"slotsPerWorker,omitempty"`
	RunPolicy         RunPolicy                                                 `json:"runPolicy,omitempty"`
	SSHAuthMountPath  string                                                    `json:"sshAuthMountPath,omitempty"`
	MPIImplementation operationv1.MPIImplementation                             `json:"mpiImplementation,omitempty"`
	MPIReplicaSpecs   map[operationv1.MPIReplicaType]*operationv1.KFReplicaSpec `json:"mpiReplicaSpecs"`
}

// PyTorchJobSpec is a desired state description of the PyTorchJob.
type PyTorchJobSpec struct {
	RunPolicy           RunPolicy                                                     `json:"runPolicy,omitempty"`
	PyTorchReplicaSpecs map[operationv1.PyTorchReplicaType]*operationv1.KFReplicaSpec `json:"pytorchReplicaSpecs"`
}

// TFJobSpec is a desired state description of the TFJob.
type TFJobSpec struct {
	RunPolicy      RunPolicy                                                `json:"runPolicy,omitempty"`
	TFReplicaSpecs map[operationv1.TFReplicaType]*operationv1.KFReplicaSpec `json:"tfReplicaSpecs"`
}

// MXJobSpec is a desired state description of the MXNetJob.
type MXJobSpec struct {
	RunPolicy      RunPolicy                                                `json:"runPolicy,omitempty"`
	JobMode        operationv1.MXJobModeType                                `json:"jobMode,omitempty"`
	MXReplicaSpecs map[operationv1.MXReplicaType]*operationv1.KFReplicaSpec `json:"mxReplicaSpecs"`
}

// XGBoostJobSpec is a desired state description of the XGBoostJob.
type XGBoostJobSpec struct {
	RunPolicy       RunPolicy                                                 `json:"runPolicy,omitempty"`
	XGBReplicaSpecs map[operationv1.XGBReplicaType]*operationv1.KFReplicaSpec `json:"xgbReplicaSpecs"`
}
