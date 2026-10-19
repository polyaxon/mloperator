package cluster

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster/daskapi"
	"github.com/polyaxon/mloperator/internal/controller/cluster/kinds"
	"github.com/polyaxon/mloperator/internal/helpers/managers"
)

func (r *ClusterReconciler) reconcileDaskClusterOp(ctx context.Context, instance *apiv1.Cluster) (ctrl.Result, error) {
	// Reconcile the underlying cluster
	if err := r.reconcileDaskCluster(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	duration, err := r.handlePastActiveDeadline(ctx, instance)
	if err != nil || duration == nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: *duration}, nil
}

func (r *ClusterReconciler) reconcileDaskCluster(ctx context.Context, instance *apiv1.Cluster) error {
	log := r.Log

	cluster, err := managers.GenerateDaskCluster(
		instance.Name,
		instance.Namespace,
		instance.Labels,
		instance.Annotations,
		instance.Termination,
		*instance.DaskClusterSpec,
	)

	if err != nil {
		log.V(1).Info("GenerateDaskCluster Error")
		return err
	}

	if err := ctrl.SetControllerReference(instance, cluster, r.Scheme); err != nil {
		log.V(1).Info("SetControllerReference Error")
		return err
	}

	// Check if the DaskCluster already exists
	foundCluster := &unstructured.Unstructured{}
	foundCluster.SetAPIVersion(kinds.DaskAPIVersion)
	foundCluster.SetKind(kinds.DaskClusterKind)
	justCreated := false
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundCluster)
	if err != nil && apierrs.IsNotFound(err) {
		// DaskCluster not found - check if we should create it or if it was deleted
		if instance.Status.IsDone() {
			// Instance is already marked as done, don't recreate
			return nil
		}

		// Check if cluster was previously created and is now missing
		// We check if there are status conditions - these only exist after the cluster has been created and started running
		// This prevents false positives on first reconcile (where StartTime exists but cluster hasn't been created yet)
		if len(instance.Status.Conditions) > 0 {
			// Cluster was created before (has conditions) but now missing
			// IMPORTANT: Don't override terminal states (Succeeded/Failed) - cluster might have been cleaned up by TTL
			if !instance.Status.IsDone() {
				// Cluster is not in terminal state but was deleted - mark as stopped
				log.Info("DaskCluster was deleted externally before completion - marking as stopped", "namespace", instance.Namespace, "name", instance.Name)
				if updated := instance.Status.LogStopped("ClusterDeleted", "The underlying DaskCluster was deleted"); updated {
					log.Info("Successfully marked DaskCluster as stopped due to external deletion")
					if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
						log.Error(statusErr, "Failed to update status after marking as stopped")
						return statusErr
					}
					_ = r.instanceSyncStatus(instance)
				}
			} else {
				// Cluster already in terminal state (Succeeded/Failed), keep it
				// This is normal - cluster was cleaned up after completion (e.g., by TTL controller)
				conditionType := "Unknown"
				if len(instance.Status.Conditions) > 0 {
					conditionType = string(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type)
				}
				log.V(1).Info("DaskCluster was cleaned up after completion", "namespace", instance.Namespace, "name", instance.Name, "status", conditionType)
			}
			return nil
		}

		// DaskCluster never existed, create it
		log.V(1).Info("Creating DaskCluster", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Create(ctx, cluster)
		if err != nil {
			if updated := instance.Status.LogWarning("OperatorCreateDaskCluster", err.Error()); updated {
				log.V(1).Info("Warning unable to create DaskCluster")
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				_ = r.instanceSyncStatus(instance)
			}
			return err
		}
		justCreated = true

		// Refetch the instance to get the latest resourceVersion
		// (Kubernetes may have modified the Cluster resource while we were creating the DaskCluster)
		if err := r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, instance); err != nil {
			log.V(1).Info("Failed to refetch Cluster after creating DaskCluster", "error", err)
			return err
		}

		instance.Status.LogStarting()
		if err := r.Status().Update(ctx, instance); err != nil {
			return err
		}
		_ = r.instanceSyncStatus(instance)
	} else if err != nil {
		return err
	}

	// Update the DaskCluster object and write the result back if there are any changes
	// Don't update if: 1) just created, 2) instance is done, 3) DaskCluster is in terminal state
	if !justCreated && !instance.Status.IsDone() && !isDaskClusterInTerminalState(foundCluster) {
		if managers.CopyUnstructuredFields(cluster, foundCluster) {
			log.V(1).Info("Updating DaskCluster", "namespace", instance.Namespace, "name", instance.Name)
			err = r.Update(ctx, foundCluster)
			if err != nil {
				return err
			}
		}
	}

	// Check the DaskCluster status
	condUpdated, err := r.reconcileDaskClusterStatus(instance, *foundCluster)
	if err != nil {
		log.V(1).Info("reconcileDaskClusterStatus Error")
		return err
	}
	if condUpdated {
		log.V(1).Info("Reconciling DaskCluster status", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Status().Update(ctx, instance)
		if err != nil {
			return err
		}
		_ = r.instanceSyncStatus(instance)
	}

	return nil
}

// getDaskReplicaFailure returns the ReplicaFailure condition of the cluster's scheduler or worker Deployments.
// The Dask operator reports Running once the scheduler Service exists, even when its Deployments can't create pods.
func (r *ClusterReconciler) getDaskReplicaFailure(instance *apiv1.Cluster) (appsv1.DeploymentCondition, bool) {
	deployments := &appsv1.DeploymentList{}
	if err := r.List(context.TODO(), deployments, client.InNamespace(instance.Namespace), client.MatchingLabels{daskClusterNameLabel: instance.Name}); err != nil {
		r.Log.Error(err, "List Dask deployments error")
		return appsv1.DeploymentCondition{}, false
	}
	for _, deployment := range deployments.Items {
		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentReplicaFailure && cond.Status == corev1.ConditionTrue {
				return cond, true
			}
		}
	}
	return appsv1.DeploymentCondition{}, false
}

// daskComponentLabel is set by the Dask operator on scheduler and worker pods.
const daskComponentLabel = "dask.org/component"

// isDaskSchedulerRunning reports whether the main container of the cluster's scheduler pod is running.
func isDaskSchedulerRunning(pods []corev1.Pod) bool {
	for _, pod := range pods {
		if pod.Labels[daskComponentLabel] == "scheduler" && managers.IsMainContainerRunning([]corev1.Pod{pod}) {
			return true
		}
	}
	return false
}

func (r *ClusterReconciler) reconcileDaskClusterStatus(instance *apiv1.Cluster, cluster unstructured.Unstructured) (bool, error) {
	now := metav1.Now()
	log := r.Log

	instanceID := instance.Labels["app.kubernetes.io/instance"]
	pods, err := managers.ListInstancePods(r.Client, instanceID, instance.Namespace)
	if err != nil {
		log.Error(err, "List pods error")
	}
	if message, found := managers.GetInvalidImageName(pods.Items); found {
		if updated := instance.Status.LogFailed(managers.InvalidImageNameReason, message); updated {
			instance.Status.CompletionTime = &now
			log.Info("Cluster image name is invalid", "Message", message)
			return true, nil
		}
		return false, nil
	}
	// Get pod status for enhanced error messages (but don't let it override cluster status)
	podStatus, reason, message := managers.HasUnschedulablePods(pods.Items)

	status, ok, unerr := unstructured.NestedFieldCopy(cluster.Object, "status")
	if !ok {
		if unerr != nil {
			log.Error(unerr, "NestedFieldCopy unstructured to status error")
			return false, nil
		}
		log.Info("NestedFieldCopy unstructured to status error",
			"err", "Status is not found in DaskCluster")
		return false, nil
	}

	statusMap := status.(map[string]interface{})
	clusterStatus := daskapi.DaskClusterStatus{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &clusterStatus)
	if err != nil {
		log.Error(err, "Convert unstructured to status error")
		return false, err
	}

	// DaskCluster uses Phase field
	switch clusterStatus.Phase {
	case daskapi.DaskClusterCreated:
		instance.Status.LogStarting()
		log.V(1).Info("Cluster Logging Status Created")
		return true, nil

	case daskapi.DaskClusterPending:
		if updated := instance.Status.LogWarning("Cluster pending", "Waiting for scheduler and workers to start"); updated {
			log.V(1).Info("Cluster Logging Status Pending")
			return true, nil
		}

	case daskapi.DaskClusterHealthChecking:
		if updated := instance.Status.LogWarning("Health checking", "Cluster is performing health checks"); updated {
			log.V(1).Info("Cluster Logging Status Health Checking")
			return true, nil
		}

	case daskapi.DaskClusterRunning:
		if cond, found := r.getDaskReplicaFailure(instance); found {
			if updated := instance.Status.LogWarning(cond.Reason, cond.Message); updated {
				log.Info("Cluster Deployment cannot create pods", "Reason", cond.Reason, "Message", cond.Message)
				return true, nil
			}
			return false, nil
		}
		// The Dask operator reports Running once the scheduler Service exists, before the scheduler starts
		if !isDaskSchedulerRunning(pods.Items) {
			if podStatus == apiv1.OperationWarning {
				if updated := instance.Status.LogWarning(reason, message); updated {
					log.Info("Cluster scheduler is not running", "Reason", reason, "Message", message)
					return true, nil
				}
			}
			return false, nil
		}
		instance.Status.LogRunning()
		log.V(1).Info("Cluster Logging Status Running")
		return true, nil

	case daskapi.DaskClusterCrashLoopBackOff:
		newMessage := apiv1.GetFailureMessage("Cluster in CrashLoopBackOff", podStatus, reason, message)
		if updated := instance.Status.LogWarning("CrashLoopBackOff", newMessage); updated {
			log.V(1).Info("Cluster Logging Status CrashLoopBackOff", "Message", newMessage)
			return true, nil
		}

	case daskapi.DaskClusterSuccessful:
		instance.Status.LogSucceeded()
		instance.Status.CompletionTime = &now
		log.V(1).Info("Cluster Logging Status Succeeded")
		return true, nil

	case daskapi.DaskClusterError:
		newMessage := apiv1.GetFailureMessage("Cluster failed", podStatus, reason, message)
		if updated := instance.Status.LogFailed(reason, newMessage); updated {
			instance.Status.CompletionTime = &now
			log.V(1).Info("Cluster Logging Status Failed", "Message", newMessage, "podStatus", podStatus, "PodMessage", message)
			return true, nil
		}

	default:
		// If DaskCluster has no clear status phase, check for severe pod issues
		// Only report warnings for truly problematic states (not transient ones like ContainersNotReady)
		if podStatus == apiv1.OperationWarning && reason != "" && reason != "ContainersNotReady" && reason != "PodInitializing" {
			log.Info("Cluster has no clear status, checking pod warnings", "Reason", reason, "message", message)
			if updated := instance.Status.LogWarning(reason, message); updated {
				log.Info("Logging Status Warning based on pod status")
				return true, nil
			}
		}
	}

	return false, nil
}

func (r *ClusterReconciler) cleanUpDaskCluster(ctx context.Context, instance *apiv1.Cluster) (ctrl.Result, error) {
	return r.handleTTL(ctx, instance)
}
