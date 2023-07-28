package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/config"
	"github.com/polyaxon/mloperator/controllers/kinds"
	"github.com/polyaxon/mloperator/controllers/utils"
)

// OperationReconciler reconciles a Operation object
type OperationReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
}

// +kubebuilder:rbac:groups=core.polyaxon.com,resources=operations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.polyaxon.com,resources=operations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.polyaxon.com,resources=operations/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=mpijobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=mpijobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=paddlejobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=paddlejobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubernetes.dask.org,resources=daskjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubernetes.dask.org,resources=daskjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ray.io,resources=rayjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=rayjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=destinationrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=destinationrules/status,verbs=get;update;patch

// Reconcile logic for OperationReconciler
func (r *OperationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("operator", req.NamespacedName)

	// Load the instance by name
	instance := &operationv1.Operation{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.V(1).Info("unable to fetch Operation", "err", err)
		return ctrl.Result{}, utils.IgnoreNotFound(err)
	}

	// Set StartTime
	if instance.Status.StartTime == nil {
		if err := r.AddStartTime(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Finalizer
	if instance.IsBeingDeleted() {
		return ctrl.Result{}, r.handleFinalizers(ctx, instance)
	} else if !instance.HasLogsFinalizer() {
		if err := r.AddLogsFinalizer(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	} else if !instance.HasNotificationsFinalizer() {
		if err := r.AddNotificationsFinalizer(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	} else if instance.IsDone() {
		return r.cleanUpOperation(ctx, instance)
	}

	// Reconcile the underlaying runtime
	return r.reconcileOperation(ctx, instance)
}

func (r *OperationReconciler) reconcileOperation(ctx context.Context, instance *operationv1.Operation) (ctrl.Result, error) {
	if instance.BatchJobSpec != nil {
		return r.reconcileJobOp(ctx, instance)
	} else if instance.ServiceSpec != nil {
		return r.reconcileServiceOp(ctx, instance)
	} else if instance.TFJobSpec != nil {
		return r.reconcileTFJobOp(ctx, instance)
	} else if instance.PytorchJobSpec != nil {
		return r.reconcilePytorchJobOp(ctx, instance)
	} else if instance.PaddleJobSpec != nil {
		return r.reconcilePaddleJobOp(ctx, instance)
	} else if instance.MXJobSpec != nil {
		return r.reconcileMXJobOp(ctx, instance)
	} else if instance.XGBoostJobSpec != nil {
		return r.reconcileXGBJobOp(ctx, instance)
	} else if instance.MPIJobSpec != nil {
		return r.reconcileMPIJobOp(ctx, instance)
	} else if instance.DaskJobSpec != nil {
		return r.reconcileDaskJobOp(ctx, instance)
	} else if instance.RayJobSpec != nil {
		return r.reconcileRayJobOp(ctx, instance)
	}
	return ctrl.Result{}, nil
}

func (r *OperationReconciler) cleanUpOperation(ctx context.Context, instance *operationv1.Operation) (ctrl.Result, error) {
	if instance.BatchJobSpec != nil {
		return r.cleanUpJob(ctx, instance)
	} else if instance.ServiceSpec != nil {
		return r.cleanUpService(ctx, instance)
	} else if instance.TFJobSpec != nil {
		return r.cleanUpTFJob(ctx, instance)
	} else if instance.PytorchJobSpec != nil {
		return r.cleanUpPytorchJob(ctx, instance)
	} else if instance.PaddleJobSpec != nil {
		return r.cleanUpPaddleJob(ctx, instance)
	} else if instance.MXJobSpec != nil {
		return r.cleanUpMXJob(ctx, instance)
	} else if instance.XGBoostJobSpec != nil {
		return r.cleanUpXGBJob(ctx, instance)
	} else if instance.MPIJobSpec != nil {
		return r.cleanUpMPIJob(ctx, instance)
	} else if instance.DaskJobSpec != nil {
		return r.cleanUpDaskJob(ctx, instance)
	} else if instance.RayJobSpec != nil {
		return r.cleanUpRayJob(ctx, instance)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager register the reconciliation logic
func (r *OperationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controllerManager := ctrl.NewControllerManagedBy(mgr).
		For(&operationv1.Operation{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: config.GetIntEnv(config.MaxConcurrentReconciles, 1)})
	controllerManager.Owns(&batchv1.Job{}).Watches(&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestForOwner{OwnerType: &batchv1.Job{}, IsController: true})
	controllerManager.Owns(&appsv1.Deployment{}).Watches(&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestForOwner{OwnerType: &appsv1.Deployment{}, IsController: true})
	controllerManager.Owns(&corev1.Service{})

	if config.GetBoolEnv(config.TFJobEnabled, false) {
		tfJob := &unstructured.Unstructured{}
		tfJob.SetAPIVersion(kinds.KFAPIVersion)
		tfJob.SetKind(kinds.TFJobKind)
		controllerManager.Owns(tfJob)
	}
	if config.GetBoolEnv(config.PytorchJobEnabled, false) {
		pytorchJob := &unstructured.Unstructured{}
		pytorchJob.SetAPIVersion(kinds.KFAPIVersion)
		pytorchJob.SetKind(kinds.PytorchJobKind)
		controllerManager.Owns(pytorchJob)
	}
	if config.GetBoolEnv(config.PaddleJobEnabled, false) {
		paddleJob := &unstructured.Unstructured{}
		paddleJob.SetAPIVersion(kinds.KFAPIVersion)
		paddleJob.SetKind(kinds.PaddleJobKind)
		controllerManager.Owns(paddleJob)
	}
	if config.GetBoolEnv(config.MPIJobEnabled, false) {
		mpiJob := &unstructured.Unstructured{}
		mpiJob.SetAPIVersion(kinds.KFAPIVersion)
		mpiJob.SetKind(kinds.MPIJobKind)
		controllerManager.Owns(mpiJob)
	}
	if config.GetBoolEnv(config.MXJobEnabled, false) {
		mxJob := &unstructured.Unstructured{}
		mxJob.SetAPIVersion(kinds.KFAPIVersion)
		mxJob.SetKind(kinds.MXJobKind)
		controllerManager.Owns(mxJob)
	}
	if config.GetBoolEnv(config.XGBoostJobEnabled, false) {
		xgBoostJob := &unstructured.Unstructured{}
		xgBoostJob.SetAPIVersion(kinds.KFAPIVersion)
		xgBoostJob.SetKind(kinds.XGBoostJobKind)
		controllerManager.Owns(xgBoostJob)
	}
	if config.GetBoolEnv(config.IstioEnabled, false) {
		istioVirtualService := &unstructured.Unstructured{}
		istioVirtualService.SetAPIVersion(kinds.IstioAPIVersion)
		istioVirtualService.SetKind(kinds.IstioVirtualServiceKind)
		controllerManager.Owns(istioVirtualService)
	}
	if config.GetBoolEnv(config.DaskJobEnabled, false) {
		daskJob := &unstructured.Unstructured{}
		daskJob.SetAPIVersion(kinds.DaskAPIVersion)
		daskJob.SetKind(kinds.DaskJobKind)
		controllerManager.Owns(daskJob)
	}
	if config.GetBoolEnv(config.RayJobEnabled, false) {
		rayJob := &unstructured.Unstructured{}
		rayJob.SetAPIVersion(kinds.RayAPIVersion)
		rayJob.SetKind(kinds.RayJobKind)
		controllerManager.Owns(rayJob)
	}
	return controllerManager.Complete(r)
}
