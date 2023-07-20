package rayapi

import (
	corev1 "k8s.io/api/core/v1"
)

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
	Template corev1.PodTemplateSpec `json:"template"`
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
	// Replicas Number of desired pods in this pod group. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	Replicas *int32 `json:"replicas"`
	// MinReplicas defaults to 1
	MinReplicas *int32 `json:"minReplicas"`
	// MaxReplicas defaults to maxInt32
	MaxReplicas *int32 `json:"maxReplicas"`
	// RayStartParams are the params of the start command: address, object-store-memory, ...
	RayStartParams map[string]string `json:"rayStartParams"`
	// Template is a pod template for the worker
	Template corev1.PodTemplateSpec `json:"template"`
	// ScaleStrategy defines which pods to remove
	ScaleStrategy ScaleStrategy `json:"scaleStrategy,omitempty"`
}

type RayClusterSpec struct {
	// HeadGroupSpecs are the spec for the head pod
	HeadGroupSpec HeadGroupSpec `json:"headGroupSpec"`
	// WorkerGroupSpecs are the specs for the worker pods
	WorkerGroupSpecs []WorkerGroupSpec `json:"workerGroupSpecs,omitempty"`
	// RayVersion is the version of ray being used. This determines the autoscaler's image version.
	RayVersion string `json:"rayVersion,omitempty"`
	// EnableInTreeAutoscaling indicates whether operator should create in tree autoscaling configs
	EnableInTreeAutoscaling *bool `json:"enableInTreeAutoscaling,omitempty"`
	// AutoscalerOptions specifies optional configuration for the Ray autoscaler.
	AutoscalerOptions      *AutoscalerOptions `json:"autoscalerOptions,omitempty"`
	HeadServiceAnnotations map[string]string  `json:"headServiceAnnotations,omitempty"`
}

type RayJobSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	Entrypoint string `json:"entrypoint"`
	// Metadata is data to store along with this job.
	Metadata map[string]string `json:"metadata,omitempty"`
	// RuntimeEnv is base64 encoded.
	RuntimeEnv string `json:"runtimeEnv,omitempty"`
	// If jobId is not set, a new jobId will be auto-generated.
	JobId string `json:"jobId,omitempty"`
	// ShutdownAfterJobFinishes will determine whether to delete the ray cluster once rayJob succeed or failed.
	ShutdownAfterJobFinishes bool `json:"shutdownAfterJobFinishes,omitempty"`
	// TTLSecondsAfterFinished is the TTL to clean up RayCluster.
	// It's only working when ShutdownAfterJobFinishes set to true.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
	// RayClusterSpec is the cluster template to run the job
	RayClusterSpec *RayClusterSpec `json:"rayClusterSpec,omitempty"`
	// clusterSelector is used to select running rayclusters by labels
	ClusterSelector map[string]string `json:"clusterSelector,omitempty"`
	// suspend specifies whether the RayJob controller should create a RayCluster instance
	// If a job is applied with the suspend field set to true,
	// the RayCluster will not be created and will wait for the transition to false.
	// If the RayCluster is already created, it will be deleted.
	// In case of transition to false a new RayCluster will be created.
	Suspend bool `json:"suspend,omitempty"`
	// SubmitterPodTemplate is the template for the pod that will run `ray job submit`.
	SubmitterPodTemplate *corev1.PodTemplateSpec `json:"submitterPodTemplate,omitempty"`
}
