package cluster

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster/kinds"
	"github.com/polyaxon/mloperator/internal/helpers/config"
	"github.com/polyaxon/mloperator/internal/helpers/utils"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
}

// +kubebuilder:rbac:groups=polyaxon.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=polyaxon.com,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=polyaxon.com,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch
// +kubebuilder:rbac:groups=kubernetes.dask.org,resources=daskclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubernetes.dask.org,resources=daskclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ray.io,resources=rayclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=rayclusters/status,verbs=get;update;patch

// Reconcile logic for ClusterReconciler
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("operator", req.NamespacedName)

	// Load the instance by name
	instance := &apiv1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Info("Unable to fetch Cluster", "err", err)
		// TODO: add check for backend status
		return ctrl.Result{}, utils.IgnoreNotFound(err)
	}

	// Set StartTime
	if instance.Status.StartTime == nil {
		if err := r.AddStartTime(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		// Refetch to get latest resourceVersion after status update
		if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
			log.Info("Failed to refetch after setting StartTime", "err", err)
			return ctrl.Result{}, err
		}
	}

	// Finalizer
	if apiv1.IsOperationBeingDeleted(instance) {
		log.V(1).Info("Operation is being deleted, remove finalizers")
		return ctrl.Result{}, r.handleFinalizers(ctx, instance)
	} else if instance.Status.IsDone() {
		log.V(1).Info("Instance is done", "CompletionTime", instance.Status.CompletionTime, "IsOperationBeingDeleted", apiv1.IsOperationBeingDeleted(instance))
		if err := r.handleFinalizers(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("Cleaning up operation")
		return r.cleanUpOperation(ctx, instance)
	} else if config.GetBoolEnv(config.EnableLogsFinalizers, false) {
		if !controllerutil.ContainsFinalizer(instance, apiv1.OperationLogsFinalizer) {
			log.V(1).Info("Adding logs finalizer", "IsDone", instance.Status.IsDone())
			if err := r.AddLogsFinalizer(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
			// Refetch to get latest resourceVersion after finalizer update
			if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
				log.Info("Failed to refetch after adding logs finalizer", "err", err)
				return ctrl.Result{}, err
			}
		}
	}

	if config.GetBoolEnv(config.EnableStatusFinalizers, false) {
		if !controllerutil.ContainsFinalizer(instance, apiv1.OperationStatusFinalizer) {
			log.V(1).Info("Adding status finalizer", "IsDone", instance.Status.IsDone())
			if err := r.AddStatusFinalizer(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
			// Refetch to get latest resourceVersion after finalizer update
			if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
				log.Info("Failed to refetch after adding status finalizer", "err", err)
				return ctrl.Result{}, err
			}
		}
	}

	// Reconcile the underlying runtime
	return r.reconcileOperation(ctx, instance)
}

func (r *ClusterReconciler) reconcileOperation(ctx context.Context, instance *apiv1.Cluster) (ctrl.Result, error) {
	if instance.DaskClusterSpec != nil {
		return r.reconcileDaskClusterOp(ctx, instance)
	} else if instance.RayClusterSpec != nil {
		return r.reconcileRayClusterOp(ctx, instance)
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) cleanUpOperation(ctx context.Context, instance *apiv1.Cluster) (ctrl.Result, error) {
	if instance.DaskClusterSpec != nil {
		return r.cleanUpDaskCluster(ctx, instance)
	} else if instance.RayClusterSpec != nil {
		return r.cleanUpRayCluster(ctx, instance)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager register the reconciliation logic
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// function to convert pod events to reconcile requests for workspaces
	mapPodToRequest := func(ctx context.Context, object client.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      object.GetLabels()[config.PolyaxonRunIdLabel],
					Namespace: object.GetNamespace(),
				},
			},
		}
	}

	// predicate function to filter pods that are labeled with the "workspace-name" label key
	predPodHasWSLabel := predicate.NewPredicateFuncs(func(object client.Object) bool {
		_, labelExists := object.GetLabels()[config.PolyaxonRunIdLabel]
		return labelExists
	})

	// Handler for unstructured cluster jobs - maps them to their owner Cluster
	// We use a custom handler instead of EnqueueRequestForOwner because the latter
	// doesn't work reliably on delete events (owner refs may not be accessible)
	mapClusterToRequestFunc := func(ctx context.Context, obj client.Object) []reconcile.Request {
		// Extract owner reference from the cluster resource (RayCluster/DaskCluster)
		for _, ownerRef := range obj.GetOwnerReferences() {
			// Check if owned by our Cluster CRD
			if ownerRef.APIVersion == "polyaxon.com/v1" && ownerRef.Kind == "Cluster" && ownerRef.Controller != nil && *ownerRef.Controller {
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name:      ownerRef.Name,
							Namespace: obj.GetNamespace(),
						},
					},
				}
			}
		}
		return []reconcile.Request{}
	}

	mapClusterToRequest := handler.EnqueueRequestsFromMapFunc(mapClusterToRequestFunc)

	// Handler for cluster-created pods - follows ownership chain: Pod -> RayCluster/DaskCluster -> Cluster
	mapClusterPodToRequest := func(ctx context.Context, object client.Object) []reconcile.Request {
		// Cluster pods are owned by the cluster job (RayCluster, DaskCluster, etc.)
		// We need to find the cluster job's owner (our Cluster) and trigger reconciliation
		pod := object.(*corev1.Pod)

		// Check if this pod is owned by a cluster job by looking at owner references
		for _, ownerRef := range pod.GetOwnerReferences() {
			// Check if owned by a Ray or Dask cluster type
			if (ownerRef.APIVersion == kinds.RayAPIVersion && ownerRef.Kind == kinds.RayClusterKind) ||
				(ownerRef.APIVersion == kinds.DaskAPIVersion && ownerRef.Kind == kinds.DaskClusterKind) {
				// The cluster job has the same name as our Cluster (set in reconcile)
				// So we can directly map to the Cluster
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name:      ownerRef.Name,
							Namespace: pod.GetNamespace(),
						},
					},
				}
			}
		}
		return []reconcile.Request{}
	}

	// Predicate for cluster pods - only trigger on meaningful changes
	predClusterPod := predicate.NewPredicateFuncs(func(object client.Object) bool {
		// Check if pod is owned by a cluster job
		for _, ownerRef := range object.GetOwnerReferences() {
			if (ownerRef.APIVersion == kinds.RayAPIVersion && ownerRef.Kind == kinds.RayClusterKind) ||
				(ownerRef.APIVersion == kinds.DaskAPIVersion && ownerRef.Kind == kinds.DaskClusterKind) {
				return true
			}
		}
		return false
	})

	controllerManager := ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Cluster{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: config.GetIntEnv(config.MaxConcurrentReconciles, 1)})
	controllerManager.Owns(&batchv1.Job{}).Owns(&corev1.Service{}).Watches(
		&corev1.Pod{},
		handler.EnqueueRequestsFromMapFunc(mapPodToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predPodHasWSLabel),
	)
	controllerManager.Owns(&appsv1.Deployment{}).Watches(
		&corev1.Pod{},
		handler.EnqueueRequestsFromMapFunc(mapPodToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predPodHasWSLabel),
	)

	// Watch cluster-created pods for status changes (e.g., unschedulable, failed, etc.)
	controllerManager.Watches(
		&corev1.Pod{},
		handler.EnqueueRequestsFromMapFunc(mapClusterPodToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predClusterPod),
	)

	// Custom predicate for cluster resources: watch spec changes, status changes, and deletions
	// GenerationChangedPredicate alone would filter out deletions since generation doesn't change on delete
	clusterResourcePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Reconcile on creation
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Reconcile on generation change (spec change)
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				return true
			}

			// Reconcile if deletion is in progress
			if !e.ObjectNew.GetDeletionTimestamp().IsZero() {
				return true
			}

			// Reconcile on status.phase changes (for DaskCluster/RayCluster)
			// This ensures we update CRD status when cluster transitions between states
			oldUnstructured, oldOk := e.ObjectOld.(*unstructured.Unstructured)
			newUnstructured, newOk := e.ObjectNew.(*unstructured.Unstructured)
			if oldOk && newOk {
				oldPhase, _, _ := unstructured.NestedString(oldUnstructured.Object, "status", "phase")
				newPhase, _, _ := unstructured.NestedString(newUnstructured.Object, "status", "phase")
				if oldPhase != newPhase && newPhase != "" {
					return true
				}

				// Also check RayCluster status.state field
				oldState, _, _ := unstructured.NestedString(oldUnstructured.Object, "status", "state")
				newState, _, _ := unstructured.NestedString(newUnstructured.Object, "status", "state")
				if oldState != newState && newState != "" {
					return true
				}
			}

			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Always reconcile on delete
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			// Reconcile on generic events
			return true
		},
	}

	// Watch DaskCluster for spec changes, status changes, and deletions
	if config.GetBoolEnv(config.DaskClusterEnabled, false) {
		daskCluster := &unstructured.Unstructured{}
		daskCluster.SetAPIVersion(kinds.DaskAPIVersion)
		daskCluster.SetKind(kinds.DaskClusterKind)
		controllerManager.Watches(
			daskCluster,
			mapClusterToRequest,
			builder.WithPredicates(clusterResourcePredicate),
		)
	}

	// Watch RayCluster for spec changes, status changes, and deletions
	if config.GetBoolEnv(config.RayClusterEnabled, false) {
		rayCluster := &unstructured.Unstructured{}
		rayCluster.SetAPIVersion(kinds.RayAPIVersion)
		rayCluster.SetKind(kinds.RayClusterKind)
		controllerManager.Watches(
			rayCluster,
			mapClusterToRequest,
			builder.WithPredicates(clusterResourcePredicate),
		)
	}

	return controllerManager.Complete(r)
}
