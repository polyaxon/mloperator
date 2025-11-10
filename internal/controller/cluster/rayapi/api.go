package rayapi

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStatus string

// https://docs.ray.io/en/latest/cluster/running-applications/job-submission/jobs-package-ref.html#jobstatus
const (
	JobStatusPending   JobStatus = "PENDING"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusStopped   JobStatus = "STOPPED"
	JobStatusSucceeded JobStatus = "SUCCEEDED"
	JobStatusFailed    JobStatus = "FAILED"
)

// The overall state of the Ray cluster.
type ClusterState string

const (
	Ready     ClusterState = "ready"
	Unhealthy ClusterState = "unhealthy"
	Failed    ClusterState = "failed"
)

// RayClusterStatus defines the observed state of RayCluster
type RayClusterStatus struct {
	State                   ClusterState       `json:"state,omitempty"`
	AvailableWorkerReplicas int32              `json:"availableWorkerReplicas,omitempty"`
	DesiredWorkerReplicas   int32              `json:"desiredWorkerReplicas,omitempty"`
	MinWorkerReplicas       int32              `json:"minWorkerReplicas,omitempty"`
	MaxWorkerReplicas       int32              `json:"maxWorkerReplicas,omitempty"`
	LastUpdateTime          *metav1.Time       `json:"lastUpdateTime,omitempty"`
	Endpoints               map[string]string  `json:"endpoints,omitempty"`
	Head                    HeadInfo           `json:"head,omitempty"`
	Reason                  string             `json:"reason,omitempty"`
	ObservedGeneration      int64              `json:"observedGeneration,omitempty"`
	Conditions              []metav1.Condition `json:"conditions,omitempty"`
}

// Helper methods to check RayCluster conditions

func (s *RayClusterStatus) IsSuspended() bool {
	return s.getConditionStatus("RayClusterSuspended") == metav1.ConditionTrue
}

func (s *RayClusterStatus) IsProvisioned() bool {
	return s.getConditionStatus("RayClusterProvisioned") == metav1.ConditionTrue
}

func (s *RayClusterStatus) IsHeadPodReady() bool {
	return s.getConditionStatus("HeadPodReady") == metav1.ConditionTrue
}

func (s *RayClusterStatus) HasReplicaFailure() bool {
	return s.getConditionStatus("ReplicaFailure") == metav1.ConditionTrue
}

func (s *RayClusterStatus) GetReplicaFailureInfo() (string, string) {
	for _, cond := range s.Conditions {
		if cond.Type == "ReplicaFailure" && cond.Status == metav1.ConditionTrue {
			return cond.Reason, cond.Message
		}
	}
	return "ReplicaFailure", "Replica failure occurred"
}

func (s *RayClusterStatus) getConditionStatus(conditionType string) metav1.ConditionStatus {
	for _, cond := range s.Conditions {
		if cond.Type == conditionType {
			return cond.Status
		}
	}
	return metav1.ConditionUnknown
}

// HeadInfo gives info about head
type HeadInfo struct {
	PodIP     string `json:"podIP,omitempty"`
	ServiceIP string `json:"serviceIP,omitempty"`
}

// HeadGroupSpec are the spec for the head pod
type HeadGroupSpec struct {
	// ServiceType is Kubernetes service type of the head service. it will be used by the workers to connect to the head pod
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
	// HeadService is the Kubernetes service of the head pod.
	HeadService *corev1.Service `json:"headService,omitempty"`
	// EnableIngress indicates whether operator should create ingress object for head service or not.
	EnableIngress *bool `json:"enableIngress,omitempty"`
	// RayStartParams are the params of the start command: node-manager-port, object-store-memory, ...
	RayStartParams map[string]string `json:"rayStartParams"`
	// Template is the exact pod template used in K8s depoyments, statefulsets, etc.
	Template corev1.PodTemplateSpec `json:"template,omitempty"`
}

// ScaleStrategy to remove workers
type ScaleStrategy struct {
	// WorkersToDelete workers to be deleted
	WorkersToDelete []string `json:"workersToDelete,omitempty"`
}

type UpscalingMode string

// AutoscalerOptions specifies optional configuration for the Ray autoscaler.
type AutoscalerOptions struct {
	// Resources specifies optional resource request and limit overrides for the autoscaler container.
	// Default values: 500m CPU request and limit. 512Mi memory request and limit.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Image optionally overrides the autoscaler's container image. This override is for provided for autoscaler testing and development.
	Image *string `json:"image,omitempty"`
	// ImagePullPolicy optionally overrides the autoscaler container's image pull policy. This override is for provided for autoscaler testing and development.
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Optional list of environment variables to set in the autoscaler container.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Optional list of sources to populate environment variables in the autoscaler container.
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
	// Optional list of volumeMounts.  This is needed for enabling TLS for the autoscaler container.
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
	// SecurityContext defines the security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// IdleTimeoutSeconds is the number of seconds to wait before scaling down a worker pod which is not using Ray resources.
	// Defaults to 60 (one minute).
	IdleTimeoutSeconds *int32 `json:"idleTimeoutSeconds,omitempty"`
	// UpscalingMode is "Conservative", "Default", or "Aggressive."
	// Conservative: Upscaling is rate-limited; the number of pending worker pods is at most the size of the Ray cluster.
	// Default: Upscaling is not rate-limited.
	// Aggressive: An alias for Default; upscaling is not rate-limited.
	UpscalingMode *UpscalingMode `json:"upscalingMode,omitempty"`
}

// WorkerGroupSpec are the specs for the worker pods
type WorkerGroupSpec struct {
	// we can have multiple worker groups, we distinguish them by name
	GroupName string `json:"groupName"`
	// Replicas is the number of desired Pods for this worker group. See https://github.com/ray-project/kuberay/pull/1443 for more details about the reason for making this field optional.
	// +kubebuilder:default:=0
	Replicas *int32 `json:"replicas,omitempty"`
	// MinReplicas denotes the minimum number of desired Pods for this worker group.
	// +kubebuilder:default:=0
	MinReplicas *int32 `json:"minReplicas"`
	// MaxReplicas denotes the maximum number of desired Pods for this worker group, and the default value is maxInt32.
	// +kubebuilder:default:=2147483647
	MaxReplicas *int32 `json:"maxReplicas"`
	// NumOfHosts denotes the number of hosts to create per replica. The default value is 1.
	// +kubebuilder:default:=1
	NumOfHosts int32 `json:"numOfHosts,omitempty"`
	// RayStartParams are the params of the start command: address, object-store-memory, ...
	RayStartParams map[string]string `json:"rayStartParams"`
	// Template is a pod template for the worker
	Template corev1.PodTemplateSpec `json:"template"`
	// ScaleStrategy defines which pods to remove
	ScaleStrategy ScaleStrategy `json:"scaleStrategy,omitempty"`
}

// RayClusterSpec defines the desired state of RayCluster
type RayClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// HeadGroupSpecs are the spec for the head pod
	HeadGroupSpec HeadGroupSpec `json:"headGroupSpec"`
	// WorkerGroupSpecs are the specs for the worker pods
	WorkerGroupSpecs []WorkerGroupSpec `json:"workerGroupSpecs,omitempty"`
	// RayVersion is used to determine the command for the Kubernetes Job managed by RayCluster
	RayVersion string `json:"rayVersion,omitempty"`
	// EnableInTreeAutoscaling indicates whether operator should create in tree autoscaling configs
	EnableInTreeAutoscaling *bool `json:"enableInTreeAutoscaling,omitempty"`
	// AutoscalerOptions specifies optional configuration for the Ray autoscaler.
	AutoscalerOptions      *AutoscalerOptions `json:"autoscalerOptions,omitempty"`
	HeadServiceAnnotations map[string]string  `json:"headServiceAnnotations,omitempty"`
	// Suspend indicates whether a RayCluster should be suspended.
	// A suspended RayCluster will have head pods and worker pods deleted.
	Suspend *bool `json:"suspend,omitempty"`
}

type JobSubmissionMode string

const (
	K8sJobMode JobSubmissionMode = "K8sJobMode" // Submit job via Kubernetes Job
	HTTPMode   JobSubmissionMode = "HTTPMode"   // Submit job via HTTP request
)
