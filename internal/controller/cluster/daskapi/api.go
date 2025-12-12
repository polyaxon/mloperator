package daskapi

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterStatus string

const (
	DaskClusterCreated          ClusterStatus = "Created"
	DaskClusterPending          ClusterStatus = "Pending"
	DaskClusterHealthChecking   ClusterStatus = "Health Checking"
	DaskClusterRunning          ClusterStatus = "Running"
	DaskClusterCrashLoopBackOff ClusterStatus = "CrashLoopBackOff"
	DaskClusterSuccessful       ClusterStatus = "Successful"
	DaskClusterError            ClusterStatus = "Error"
)

type WorkerSpec struct {
	Metadata *metav1.ObjectMeta `json:"metadata,omitempty"`
	Replicas int                `json:"replicas"`
	Spec     corev1.PodSpec     `json:"spec"`
}

type SchedulerSpec struct {
	Metadata *metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec     corev1.PodSpec     `json:"spec"`
	Service  corev1.ServiceSpec `json:"service"`
}

type DaskClusterSpec struct {
	Worker    WorkerSpec    `json:"worker"`
	Scheduler SchedulerSpec `json:"scheduler"`
}

// DaskClusterStatus describes the current status of a Dask Cluster
type DaskClusterStatus struct {
	Phase ClusterStatus `json:"phase,omitempty"`
}

type DaskCluster struct {
	Spec DaskClusterSpec `json:"spec"`
}

// DaskAutoscalerSpec defines the desired state of a DaskAutoscaler
type DaskAutoscalerSpec struct {
	// Cluster is the name of the DaskCluster to autoscale
	Cluster string `json:"cluster"`
	// Minimum is the minimum number of workers
	Minimum int `json:"minimum"`
	// Maximum is the maximum number of workers
	Maximum int `json:"maximum"`
}

// DaskAutoscaler is the Schema for the daskautoscalers API
type DaskAutoscaler struct {
	Spec DaskAutoscalerSpec `json:"spec"`
}
