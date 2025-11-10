package job

import (
	"context"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
	"github.com/polyaxon/mloperator/internal/helpers/config"
	"github.com/polyaxon/mloperator/internal/helpers/utils"
)

// JobReconciler reconciles a Operation object
type JobReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
}

// +kubebuilder:rbac:groups=polyaxon.com,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=polyaxon.com,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=polyaxon.com,resources=jobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch

// Reconcile logic for JobReconciler
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("operator", req.NamespacedName)

	// Load the instance by name
	instance := &apiv1.Job{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.V(1).Info("unable to fetch Job", "err", err)
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

func (r *JobReconciler) reconcileOperation(ctx context.Context, instance *apiv1.Job) (ctrl.Result, error) {
	return r.reconcileJobOp(ctx, instance)
}

func (r *JobReconciler) cleanUpOperation(ctx context.Context, instance *apiv1.Job) (ctrl.Result, error) {
	return r.cleanUpJob(ctx, instance)
}

// SetupWithManager register the reconciliation logic
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {

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

	controllerManager := ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Job{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: config.GetIntEnv(config.MaxConcurrentReconciles, 1)})
	controllerManager.Owns(&batchv1.Job{}).Watches(
		&corev1.Pod{},
		handler.EnqueueRequestsFromMapFunc(mapPodToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predPodHasWSLabel),
	)
	return controllerManager.Complete(r)
}
