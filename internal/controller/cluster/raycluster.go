package cluster

import (
	"context"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster/kinds"
	"github.com/polyaxon/mloperator/internal/controller/cluster/rayapi"
	"github.com/polyaxon/mloperator/internal/helpers/managers"
)

func (r *ClusterReconciler) reconcileRayClusterOp(ctx context.Context, instance *apiv1.Cluster) (ctrl.Result, error) {
	// Reconcile the underlaying job
	return ctrl.Result{}, r.reconcileRayCluster(ctx, instance)
}

func (r *ClusterReconciler) reconcileRayCluster(ctx context.Context, instance *apiv1.Cluster) error {
	log := r.Log

	job, err := managers.GenerateRayCluster(
		instance.Name,
		instance.Namespace,
		instance.Labels,
		instance.Annotations,
		instance.Termination,
		*instance.RayClusterSpec,
	)

	if err != nil {
		log.V(1).Info("GenerateRayCluster Error")
		return err
	}

	if err := ctrl.SetControllerReference(instance, job, r.Scheme); err != nil {
		log.V(1).Info("SetControllerReference Error")
		return err
	}

	// Check if the Job already exists
	foundJob := &unstructured.Unstructured{}
	foundJob.SetAPIVersion(kinds.RayAPIVersion)
	foundJob.SetKind(kinds.RayClusterKind)
	justCreated := false
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundJob)
	if err != nil && apierrs.IsNotFound(err) {
		// Job not found - check if we should create it or if it was deleted
		if instance.Status.IsDone() {
			// Instance is already marked as done, don't recreate
			return nil
		}

		// Check if job was previously created and is now missing
		// We check if there are status conditions - these only exist after the job has been created and started running
		// This prevents false positives on first reconcile (where StartTime exists but job hasn't been created yet)
		if len(instance.Status.Conditions) > 0 {
			// Job was created before (has conditions) but now missing
			// IMPORTANT: Don't override terminal states (Succeeded/Failed) - job might have been cleaned up by TTL
			if !instance.Status.IsDone() {
				// Job is not in terminal state but was deleted - mark as stopped
				log.Info("RayCluster was deleted externally before completion - marking as stopped", "namespace", instance.Namespace, "name", instance.Name)
				if updated := instance.Status.LogStopped("JobDeleted", "The underlying RayCluster was deleted"); updated {
					log.Info("Successfully marked RayCluster as stopped due to external deletion")
					if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
						return statusErr
					}
					r.instanceSyncStatus(instance)
				}
			} else {
				// Job already in terminal state (Succeeded/Failed), keep it
				conditionType := "Unknown"
				if len(instance.Status.Conditions) > 0 {
					conditionType = string(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type)
				}
				log.V(1).Info("RayCluster was cleaned up after completion", "namespace", instance.Namespace, "name", instance.Name, "status", conditionType)
			}
			return nil
		}

		// RayCluster never existed, create it
		log.V(1).Info("Creating RayCluster", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Create(ctx, job)
		if err != nil {
			if updated := instance.Status.LogWarning("OperatorCreateRayCluster", err.Error()); updated {
				log.V(1).Info("Warning unable to create RayCluster")
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				r.instanceSyncStatus(instance)
			}
			return err
		}
		justCreated = true

		// Refetch the instance to get the latest resourceVersion
		// (Kubernetes may have modified the Cluster resource while we were creating the RayCluster)
		if err := r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, instance); err != nil {
			log.V(1).Info("Failed to refetch Cluster after creating RayCluster", "error", err)
			return err
		}

		instance.Status.LogStarting()
		err = r.Status().Update(ctx, instance)
		r.instanceSyncStatus(instance)
	} else if err != nil {
		return err
	}

	// Update the job object and write the result back if there are any changes
	// Don't update if: 1) just created, 2) instance is done, 3) RayCluster is in terminal state
	if !justCreated && !instance.Status.IsDone() && !isRayClusterInTerminalState(foundJob) {
		if managers.CopyUnstructuredFields(job, foundJob) {
			log.V(1).Info("Updating RayCluster", "namespace", instance.Namespace, "name", instance.Name)
			err = r.Update(ctx, foundJob)
			if err != nil {
				return err
			}
		}
	}

	// Check the job status
	condUpdated, err := r.reconcileRayClusterStatus(instance, *foundJob)
	if err != nil {
		log.V(1).Info("reconcileRayClusterStatus Error")
		return err
	}
	if condUpdated {
		log.V(1).Info("Reconciling RayCluster status", "namespace", instance.Namespace, "name", instance.Name)
		err = r.Status().Update(ctx, instance)
		if err != nil {
			return err
		}
		r.instanceSyncStatus(instance)
	}

	return nil
}

func (r *ClusterReconciler) reconcileRayClusterStatus(instance *apiv1.Cluster, job unstructured.Unstructured) (bool, error) {
	now := metav1.Now()
	log := r.Log

	// Check the pods
	instanceID, ok := instance.ObjectMeta.Labels["app.kubernetes.io/instance"]
	podStatus, reason, message := managers.HasUnschedulablePods(r.Client, instanceID, instance.Namespace)
	if podStatus == apiv1.OperationWarning {
		log.V(1).Info("Service has unschedulable pod(s)", "Reason", reason, "message", message)
		if updated := instance.Status.LogWarning(reason, message); updated {
			log.V(1).Info("Service Logging Status Warning")
			return true, nil
		}
		return false, nil
	}

	status, ok, unerr := unstructured.NestedFieldCopy(job.Object, "status")
	if !ok {
		if unerr != nil {
			log.Error(unerr, "NestedFieldCopy unstructured to status error")
			return false, nil
		}
		log.Info("NestedFieldCopy unstructured to status error",
			"err", "Status is not found in job")
		return false, nil
	}

	statusMap := status.(map[string]interface{})
	clusterStatus := rayapi.RayClusterStatus{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, &clusterStatus)
	if err != nil {
		log.Error(err, "Convert unstructured to status error")
		return false, err
	}

	// Check if cluster is suspended
	if clusterStatus.IsSuspended() {
		if updated := instance.Status.LogWarning("Suspended", "RayCluster is suspended"); updated {
			log.V(1).Info("Cluster Logging Status Suspended")
			return true, nil
		}
		return false, nil
	}

	// Check for replica failures
	if clusterStatus.HasReplicaFailure() {
		failureReason, failureMessage := clusterStatus.GetReplicaFailureInfo()
		newMessage := apiv1.GetFailureMessage(failureMessage, podStatus, reason, message)
		if updated := instance.Status.LogFailed(failureReason, newMessage); updated {
			instance.Status.CompletionTime = &now
			log.V(1).Info("Cluster Logging Status Failed due to replica failure", "Message", newMessage)
			return true, nil
		}
		return false, nil
	}

	// Check if head pod is ready and cluster is provisioned
	if clusterStatus.IsHeadPodReady() && clusterStatus.IsProvisioned() {
		instance.Status.LogRunning()
		log.V(1).Info("Cluster Logging Status Running (head pod ready and provisioned)")
		return true, nil
	}

	// Check if head pod is ready but not fully provisioned yet
	if clusterStatus.IsHeadPodReady() {
		if updated := instance.Status.LogWarning("Provisioning", "Head pod is ready, waiting for workers"); updated {
			log.V(1).Info("Cluster Logging Status Provisioning")
			return true, nil
		}
		return false, nil
	}

	return false, nil
}

func (r *ClusterReconciler) cleanUpRayCluster(ctx context.Context, instance *apiv1.Cluster) (ctrl.Result, error) {
	return r.handleTTL(ctx, instance)
}
