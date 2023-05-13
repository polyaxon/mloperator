package managers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
)

// generateKFReplica generates a new ReplocaSpec
func generateKFReplica(replicSpec operationv1.KFReplicaSpec) *operationv1.KFReplicaSpec {
	return &operationv1.KFReplicaSpec{
		Replicas:      replicSpec.Replicas,
		RestartPolicy: replicSpec.RestartPolicy,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}},
			Spec:       replicSpec.Template.Spec,
		},
	}
}
