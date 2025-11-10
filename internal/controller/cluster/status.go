package cluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster/daskapi"
	"github.com/polyaxon/mloperator/internal/controller/cluster/rayapi"
)

// isRayClusterInTerminalState checks if a RayCluster has reached a terminal state (Succeeded or Failed)
func isRayClusterInTerminalState(job *unstructured.Unstructured) bool {
	status, ok, err := unstructured.NestedFieldCopy(job.Object, "status")
	if !ok || err != nil {
		return false
	}

	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return false
	}

	clusterStatus := rayapi.RayClusterStatus{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &clusterStatus)
	if err != nil {
		return false
	}

	// RayCluster uses State field (ready, unhealthy, failed)
	// Terminal state is when cluster has failed
	return clusterStatus.State == rayapi.Failed
}

// isDaskClusterInTerminalState checks if a DaskCluster has reached a terminal state
func isDaskClusterInTerminalState(cluster *unstructured.Unstructured) bool {
	status, ok, err := unstructured.NestedFieldCopy(cluster.Object, "status")
	if !ok || err != nil {
		return false
	}

	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return false
	}

	clusterStatus := daskapi.DaskClusterStatus{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &clusterStatus)
	if err != nil {
		return false
	}

	// DaskCluster uses Phase field
	// Terminal states are Successful and Error
	return clusterStatus.Phase == daskapi.DaskClusterSuccessful || clusterStatus.Phase == daskapi.DaskClusterError
}

// AddStartTime Adds starttime field by the reconciler
func (r *ClusterReconciler) AddStartTime(ctx context.Context, instance *apiv1.Cluster) error {
	if instance.Status.StartTime != nil {
		return nil
	}

	now := metav1.Now()
	log := r.Log

	log.V(1).Info("Setting StartTime", "Operation", instance.Name)
	instance.Status.StartTime = &now
	return r.Status().Update(ctx, instance)
}
