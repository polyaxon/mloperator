package managers

import (
	"reflect"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/polyaxon/mloperator/controllers/utils"
)

// CopyJobFields copies the owned fields from one Job to another
// Returns true if the fields copied from don't match to.
func CopyJobFields(from, to *batchv1.Job) bool {
	requireUpdate := false
	for k, v := range to.Labels {
		if from.Labels[k] != v {
			requireUpdate = true
		}
	}
	to.Labels = from.Labels

	if to.Spec.ActiveDeadlineSeconds != from.Spec.ActiveDeadlineSeconds {
		to.Spec.ActiveDeadlineSeconds = from.Spec.ActiveDeadlineSeconds
		requireUpdate = true
	}

	if to.Spec.BackoffLimit != from.Spec.BackoffLimit {
		to.Spec.BackoffLimit = from.Spec.BackoffLimit
		requireUpdate = true
	}

	if to.Spec.TTLSecondsAfterFinished != from.Spec.TTLSecondsAfterFinished {
		to.Spec.TTLSecondsAfterFinished = from.Spec.TTLSecondsAfterFinished
		requireUpdate = true
	}

	if !reflect.DeepEqual(to.Spec.Template.Spec, from.Spec.Template.Spec) {
		requireUpdate = true
		to.Spec.Template.Spec = from.Spec.Template.Spec
	}

	return requireUpdate
}

// IsJobSucceeded return true if job is running
func IsJobSucceeded(jc batchv1.JobCondition) bool {
	return jc.Type == batchv1.JobComplete && jc.Status == corev1.ConditionTrue
}

// IsJobFailed return true if job is running
func IsJobFailed(jc batchv1.JobCondition) bool {
	return jc.Type == batchv1.JobFailed && jc.Status == corev1.ConditionTrue
}

// GenerateJob returns a batch job given a OperationSpec
func GenerateJob(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	backoffLimit *int32,
	activeDeadlineSeconds *int64,
	ttlSecondsAfterFinished *int32,
	podSpec corev1.PodSpec,
) *batchv1.Job {
	if podSpec.RestartPolicy == "" {
		podSpec.RestartPolicy = utils.DefaultRestartPolicy
	}
	l := make(map[string]string)
	for k, v := range labels {
		l[k] = v
	}
	a := make(map[string]string)
	for k, v := range annotations {
		a[k] = v
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            utils.GetBackoffLimit(backoffLimit),
			ActiveDeadlineSeconds:   activeDeadlineSeconds,
			TTLSecondsAfterFinished: utils.GetTTL(ttlSecondsAfterFinished),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: l, Annotations: a},
				Spec:       podSpec,
			},
		},
	}

	return job
}
