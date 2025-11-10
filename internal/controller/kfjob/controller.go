package kfjob

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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/kfjob/kinds"
	"github.com/polyaxon/mloperator/internal/helpers/config"
	"github.com/polyaxon/mloperator/internal/helpers/utils"
)

// KfJobReconciler reconciles a Operation object
type KfJobReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
}

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch
// +kubebuilder:rbac:groups=polyaxon.com,resources=kfjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=polyaxon.com,resources=kfjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=polyaxon.com,resources=kfjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=mpijobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=mpijobs/status,verbs=get;update;patch

// Reconcile logic for KfJobReconciler
func (r *KfJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("operator", req.NamespacedName)

	// Load the instance by name
	instance := &apiv1.KfJob{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.V(1).Info("unable to fetch KfJob", "err", err)
		// TODO: add check for backend status
		return ctrl.Result{}, utils.IgnoreNotFound(err)
	}

	// Set StartTime
	if instance.Status.StartTime == nil {
		if err := r.AddStartTime(ctx, instance); err != nil {
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
		}
	}

	if config.GetBoolEnv(config.EnableStatusFinalizers, false) {
		if !controllerutil.ContainsFinalizer(instance, apiv1.OperationStatusFinalizer) {
			log.V(1).Info("Adding status finalizer", "IsDone", instance.Status.IsDone())
			if err := r.AddStatusFinalizer(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Reconcile the underlying runtime
	return r.reconcileOperation(ctx, instance)
}

func (r *KfJobReconciler) reconcileOperation(ctx context.Context, instance *apiv1.KfJob) (ctrl.Result, error) {
	if instance.TFJobSpec != nil {
		return r.reconcileTFJobOp(ctx, instance)
	} else if instance.PytorchJobSpec != nil {
		return r.reconcilePytorchJobOp(ctx, instance)
	} else if instance.MPIJobSpec != nil {
		return r.reconcileMPIJobOp(ctx, instance)
	}
	return ctrl.Result{}, nil
}

func (r *KfJobReconciler) cleanUpOperation(ctx context.Context, instance *apiv1.KfJob) (ctrl.Result, error) {
	if instance.TFJobSpec != nil {
		return r.cleanUpTFJob(ctx, instance)
	} else if instance.PytorchJobSpec != nil {
		return r.cleanUpPytorchJob(ctx, instance)
	} else if instance.MPIJobSpec != nil {
		return r.cleanUpMPIJob(ctx, instance)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager register the reconciliation logic
func (r *KfJobReconciler) SetupWithManager(mgr ctrl.Manager) error {

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

	// Handler for unstructured Kubeflow jobs - maps them to their owner KfJob
	mapKfJobToRequest := handler.EnqueueRequestForOwner(
		mgr.GetScheme(),
		mgr.GetRESTMapper(),
		&apiv1.KfJob{},
		handler.OnlyControllerOwner(),
	)

	// Handler for Kubeflow-created pods - follows ownership chain: Pod -> KfJob -> KfJob
	mapKfPodToRequest := func(ctx context.Context, object client.Object) []reconcile.Request {
		// Kubeflow pods are owned by the Kubeflow job (TFJob, PyTorchJob, etc.)
		// We need to find the Kubeflow job's owner (our KfJob) and trigger reconciliation
		pod := object.(*corev1.Pod)

		// Check if this pod is owned by a Kubeflow job by looking at owner references
		for _, ownerRef := range pod.GetOwnerReferences() {
			// Check if owned by a Kubeflow job type
			if ownerRef.APIVersion == kinds.KFAPIVersion {
				switch ownerRef.Kind {
				case kinds.TFJobKind, kinds.PytorchJobKind, kinds.MPIJobKind:
					// The Kubeflow job has the same name as our KfJob (set in reconcile)
					// So we can directly map to the KfJob
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
		}
		return []reconcile.Request{}
	}

	// Predicate for Kubeflow pods - only trigger on meaningful changes
	predKfPod := predicate.NewPredicateFuncs(func(object client.Object) bool {
		// Check if pod is owned by a Kubeflow job
		for _, ownerRef := range object.GetOwnerReferences() {
			if ownerRef.APIVersion == kinds.KFAPIVersion {
				switch ownerRef.Kind {
				case kinds.TFJobKind, kinds.PytorchJobKind, kinds.MPIJobKind:
					return true
				}
			}
		}
		return false
	})

	controllerManager := ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KfJob{}).
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

	// Watch Kubeflow-created pods for status changes (e.g., unschedulable, failed, etc.)
	controllerManager.Watches(
		&corev1.Pod{},
		handler.EnqueueRequestsFromMapFunc(mapKfPodToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predKfPod),
	)

	// Always watch TFJob for status changes (unconditional)
	if config.GetBoolEnv(config.TFJobEnabled, false) {
		tfJob := &unstructured.Unstructured{}
		tfJob.SetAPIVersion(kinds.KFAPIVersion)
		tfJob.SetKind(kinds.TFJobKind)
		controllerManager.Watches(
			tfJob,
			mapKfJobToRequest,
			builder.WithPredicates(predicate.GenerationChangedPredicate{}, predicate.ResourceVersionChangedPredicate{}),
		)
	}

	// Always watch PyTorchJob for status changes (unconditional)
	if config.GetBoolEnv(config.PytorchJobEnabled, false) {
		pytorchJob := &unstructured.Unstructured{}
		pytorchJob.SetAPIVersion(kinds.KFAPIVersion)
		pytorchJob.SetKind(kinds.PytorchJobKind)
		controllerManager.Watches(
			pytorchJob,
			mapKfJobToRequest,
			builder.WithPredicates(predicate.GenerationChangedPredicate{}, predicate.ResourceVersionChangedPredicate{}),
		)
	}

	// Always watch MPIJob for status changes (unconditional)
	if config.GetBoolEnv(config.MPIJobEnabled, false) {
		mpiJob := &unstructured.Unstructured{}
		mpiJob.SetAPIVersion(kinds.KFAPIVersion)
		mpiJob.SetKind(kinds.MPIJobKind)
		controllerManager.Watches(
			mpiJob,
			mapKfJobToRequest,
			builder.WithPredicates(predicate.GenerationChangedPredicate{}, predicate.ResourceVersionChangedPredicate{}),
		)
	}

	return controllerManager.Complete(r)
}
