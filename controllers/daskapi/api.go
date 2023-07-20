package daskapi

import (
	corev1 "k8s.io/api/core/v1"
)

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
