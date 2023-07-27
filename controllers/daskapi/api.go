package daskapi

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStatus string

const (
	DaskJobCreated        JobStatus = "JobCreated"
	DaskJobClusterCreated JobStatus = "ClusterCreated"
	DaskJobRunning        JobStatus = "Running"
	DaskJobSuccessful     JobStatus = "Successful"
	DaskJobFailed         JobStatus = "Failed"
)

// DaskJobStatus describes the current status of a Dask Job
type DaskJobStatus struct {
	// The name of the cluster the job is executed on
	ClusterName string `json:"clusterName,omitempty"`
	// The time the job runner pod changed to either Successful or Failing
	EndTime metav1.Time `json:"endTime,omitempty"`
	// The name of the job-runner pod
	JobRunnerPodName string `json:"jobRunnerPodName,omitempty"`
	// JobStatus describes the current status of the job
	JobStatus JobStatus `json:"jobStatus"`
	// Start time records the time the job-runner pod changed into a `running` state
	StartTime metav1.Time `json:"startTime"`
}

type WorkerSpec struct {
	Replicas int            `json:"replicas"`
	Spec     corev1.PodSpec `json:"spec"`
}

type SchedulerSpec struct {
	Spec    corev1.PodSpec     `json:"spec"`
	Service corev1.ServiceSpec `json:"service"`
}

type DaskClusterSpec struct {
	Worker    WorkerSpec    `json:"worker"`
	Scheduler SchedulerSpec `json:"scheduler"`
}

type DaskCluster struct {
	Spec DaskClusterSpec `json:"spec"`
}

type JobSpec struct {
	Spec corev1.PodSpec `json:"spec"`
}

type DaskJobSpec struct {
	Job     JobSpec     `json:"job"`
	Cluster DaskCluster `json:"cluster"`
}
