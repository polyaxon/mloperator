package service

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
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
	"github.com/polyaxon/mloperator/internal/controller/service/kinds"
	"github.com/polyaxon/mloperator/internal/helpers/config"
	"github.com/polyaxon/mloperator/internal/helpers/utils"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
}

// +kubebuilder:rbac:groups=polyaxon.com,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=polyaxon.com,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=polyaxon.com,resources=services/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=destinationrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=destinationrules/status,verbs=get;update;patch

// Reconcile logic for ServiceReconciler
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("service", req.NamespacedName)

	// Load the instance by name
	instance := &apiv1.Service{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.V(1).Info("unable to fetch Service", "err", err)
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
		return r.cleanUpService(ctx, instance)
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
	return r.reconcileService(ctx, instance)
}

// SetupWithManager register the reconciliation logic
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {

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
		For(&apiv1.Service{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: config.GetIntEnv(config.MaxConcurrentReconciles, 1)})
	controllerManager.Owns(&corev1.Service{}).Watches(
		&corev1.Pod{},
		handler.EnqueueRequestsFromMapFunc(mapPodToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predPodHasWSLabel),
	)
	controllerManager.Owns(&appsv1.Deployment{}).Watches(
		&corev1.Pod{},
		handler.EnqueueRequestsFromMapFunc(mapPodToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predPodHasWSLabel),
	)
	if config.GetBoolEnv(config.IstioEnabled, false) {
		istioVirtualService := &unstructured.Unstructured{}
		istioVirtualService.SetAPIVersion(kinds.IstioAPIVersion)
		istioVirtualService.SetKind(kinds.IstioVirtualServiceKind)
		controllerManager.Owns(istioVirtualService)
	}
	return controllerManager.Complete(r)
}
