package v1

import (
	batchv1 "k8s.io/api/batch/v1"
)

// TODO: integerate this it when https://github.com/kubernetes/kubernetes/issues/28486 has been fixed
// Optional number of failed pods to retain. This will be especially good for when restart is True since the underlaying pods will disapear.

// TerminationSpec defines the desired termination specification for handliong timeout, retries, and ttl
// +k8s:openapi-gen=true
type TerminationSpec struct {
	// Specifies the number of retries before marking this job failed.
	// Defaults to 0
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty" default:"1" protobuf:"varint,1,opt,name=backoffLimit"`

	// Specifies the duration (in seconds) since startTime during which the job can remain active
	// before it is terminated. Must be a positive integer.
	// This setting applies only to pods where restartPolicy is OnFailure or Always.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"varint,2,opt,name=activeDeadlineSeconds"`

	// ttlSecondsAfterFinished limits the lifetime of a Job that has finished
	// execution (either Complete or Failed). If this field is set,
	// ttlSecondsAfterFinished after the Job finishes, it is eligible to be
	// automatically deleted. When the Job is being deleted, its lifecycle
	// guarantees (e.g. finalizers) will be honored. If this field is unset,
	// the Job won't be automatically deleted. If this field is set to zero,
	// the Job becomes eligible to be deleted immediately after it finishes.
	// This field is alpha-level and is only honored by servers that enable the
	// TTLAfterFinished feature.
	// TODO: (Cleanup logic once kubernetes adds the cleanup controller)
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty" protobuf:"varint,3,opt,name=ttlSecondsAfterFinished"`

	// Culling defines the culling specification for the service
	// +optional
	Culling *CullingSpec `json:"culling,omitempty" protobuf:"bytes,4,opt,name=culling"`

	// Probe defines the activity probe for the service
	// +optional
	Probe *ActivityProbe `json:"probe,omitempty" protobuf:"bytes,5,opt,name=probe"`

	// PodFailurePolicy defines fine-grained rules for how pod failures should be handled.
	// Requires Kubernetes v1.25+ with PodDisruptionConditions and JobPodFailurePolicy feature gates enabled.
	// +optional
	PodFailurePolicy *batchv1.PodFailurePolicy `json:"podFailurePolicy,omitempty" protobuf:"bytes,6,opt,name=podFailurePolicy"`
}

// CullingSpec defines the configuration for culling idle services
// +k8s:openapi-gen=true
type CullingSpec struct {
	// Timeout is the duration in seconds that the service needs to be idle before it is culled
	// +optional
	Timeout *int32 `json:"timeout,omitempty" protobuf:"varint,1,opt,name=timeout"`
}

// ActivityProbe defines the configuration for checking activity
// +k8s:openapi-gen=true
type ActivityProbe struct {
	// Exec specifies the action to take.
	// +optional
	Exec *ActivityProbeExec `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`

	// Http specifies the http configuration.
	// +optional
	Http *ActivityProbeHttp `json:"http,omitempty" protobuf:"bytes,2,opt,name=http"`
}

// ActivityProbeExec defines the configuration for exec probe
// +k8s:openapi-gen=true
type ActivityProbeExec struct {
	// Command is the command line to execute inside the container, the working directory for the
	// command  is root ('/') in the container's filesystem. The command is simply exec'd, it is
	// not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use
	// a shell, you need to explicitly call out to that shell.
	// Exit status of 0 is treated as live/healthy and non-zero is unhealthy.
	// +optional
	Command []string `json:"command,omitempty" protobuf:"bytes,1,rep,name=command"`
}

// ActivityProbeHttp defines the configuration for http probe
// +k8s:openapi-gen=true
type ActivityProbeHttp struct {
	// Path is the path to the http server
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`

	// Port is the port to the http server
	// +optional
	Port int32 `json:"port,omitempty" protobuf:"varint,2,opt,name=port"`
}
