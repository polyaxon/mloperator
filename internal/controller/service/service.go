package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/helpers/managers"
)

// Reconcile logic for Operation
func (r *ServiceReconciler) reconcileService(ctx context.Context, instance *apiv1.Service) (ctrl.Result, error) {
	// log := r.Log

	ports := managers.GetPodPorts(instance.ServiceSpec.Template.Spec, managers.DefaultTargetPort)
	if instance.ServiceSpec.Ports != nil {
		ports = instance.ServiceSpec.Ports
	}

	// Reconcile the underlaying deployment
	if err := r.reconcileDeployment(ctx, instance, ports); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile the underlaying service
	if err := r.reconcileBaseService(ctx, instance, ports, instance.ServiceSpec.IsExternal); err != nil {
		return ctrl.Result{}, err
	}

	if duration, err := r.handlePastActiveDeadline(ctx, instance); err != nil || duration != nil {
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true, RequeueAfter: *duration}, nil
	}

	if duration, err := r.handleCulling(ctx, instance); err != nil || duration != nil {
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true, RequeueAfter: *duration}, nil
	}

	if instance.Status.IsWarning() {
		if err := r.handleServiceBackoffLimit(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		// log.V(1).Info("service has warning", "Reschdule check in", 30)
		// return ctrl.Result{Requeue: true, RequeueAfter: time.Second * time.Duration(30)}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) reconcileDeployment(ctx context.Context, instance *apiv1.Service, ports []int32) error {
	log := r.Log

	replicas := managers.GetReplicas(managers.DefaultServiceReplicas, *instance.ServiceSpec)
	deployment, err := managers.GenerateDeployment(
		instance.Name,
		instance.Namespace,
		instance.Labels,
		instance.Annotations,
		ports,
		replicas,
		instance.ServiceSpec.Template.Spec,
	)
	if err != nil {
		return err
	}
	log.V(1).Info("SetControllerReference")
	if err := ctrl.SetControllerReference(instance, deployment, r.Scheme); err != nil {
		return err
	}
	// Check if the Deployment already exists
	foundDeployment := &appsv1.Deployment{}
	justCreated := false
	log.V(1).Info("Get Service deployment")
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if instance.Status.IsDone() {
		return nil
	}
	if err != nil && apierrs.IsNotFound(err) {
		log.V(1).Info("Creating Service Deployment", "namespace", deployment.Namespace, "name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			if updated := instance.Status.LogWarning("OperatorCreateDeployment", err.Error()); updated {
				log.V(1).Info("Warning unable to create Deployment")
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				_ = r.instanceSyncStatus(instance)
			}
			return err
		}
		justCreated = true
		instance.Status.LogStarting()
		err = r.Status().Update(ctx, instance)
		_ = r.instanceSyncStatus(instance)
	} else if err != nil {
		return err
	}
	// Update the deployment object and write the result back if there are any changes
	if !justCreated && !instance.Status.IsDone() && managers.CopyDeploymentFields(deployment, foundDeployment) {
		log.V(1).Info("Updating Service Deployment", "namespace", deployment.Namespace, "name", deployment.Name)
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			return err
		}
	}

	// Check the deployment status
	if condUpdated := r.reconcileDeploymentStatus(instance, *foundDeployment); condUpdated {
		log.V(1).Info("Reconciling Service status", "namespace", deployment.Namespace, "name", deployment.Name)
		err = r.Status().Update(ctx, instance)
		if err != nil {
			return err
		}
		_ = r.instanceSyncStatus(instance)
	}

	return nil
}

func (r *ServiceReconciler) reconcileDeploymentStatus(instance *apiv1.Service, deployment appsv1.Deployment) bool {
	log := r.Log

	// Check the pods
	instanceID, _ := instance.ObjectMeta.Labels["app.kubernetes.io/instance"]
	podStatus, reason, message := managers.HasUnschedulablePods(r.Client, instanceID, instance.Namespace)
	if podStatus == apiv1.OperationWarning || podStatus == apiv1.OperationFailed {
		log.V(1).Info("Service has unschedulable pod(s)", "Reason", reason, "message", message)
		if updated := instance.Status.LogWarning(reason, message); updated {
			log.V(1).Info("Service Logging Status Warning")
			return true
		}
		return false
	}

	if len(deployment.Status.Conditions) == 0 {
		log.V(1).Info("Service No Conditions")
		return false
	}

	newDeploymentCond := deployment.Status.Conditions[len(deployment.Status.Conditions)-1]

	if managers.IsDeploymentWarning(deployment.Status, newDeploymentCond) {
		instance.Status.LogWarning(newDeploymentCond.Reason, newDeploymentCond.Message)
		log.V(1).Info("Service Logging Status Warning")
		return true
	}

	if managers.IsDeploymentRunning(deployment.Status, newDeploymentCond) {
		instance.Status.LogRunning()
		log.V(1).Info("Service Logging Status Running")
		return true
	}
	return false
}

func (r *ServiceReconciler) reconcileBaseService(ctx context.Context, instance *apiv1.Service, ports []int32, isExternal bool) error {
	log := r.Log

	name := instance.Name
	if isExternal {
		name += "-ext"
	}
	service := managers.GenerateService(name, instance.Namespace, instance.Labels, instance.Annotations, ports)
	if err := ctrl.SetControllerReference(instance, service, r.Scheme); err != nil {
		log.V(1).Info("generateService Error")
		return err
	}
	// Check if the Service already exists
	foundService := &corev1.Service{}
	justCreated := false
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && apierrs.IsNotFound(err) {
		log.V(1).Info("Creating Service", "namespace", service.Namespace, "name", service.Name)
		err = r.Create(ctx, service)
		if err != nil {
			if updated := instance.Status.LogWarning("OperatorCreateService", err.Error()); updated {
				log.V(1).Info("Warning unable to create Service")
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return statusErr
				}
				_ = r.instanceSyncStatus(instance)
			}
			return err
		}
		justCreated = true
	} else if err != nil {
		return err
	}
	// Update the foundService object and write the result back if there are any changes
	if !justCreated && managers.CopyServiceFields(service, foundService) {
		log.V(1).Info("Updating Service\n", "namespace", service.Namespace, "name", service.Name)
		err = r.Update(ctx, foundService)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceReconciler) cleanUpService(ctx context.Context, instance *apiv1.Service) (ctrl.Result, error) {
	return r.handleTTL(ctx, instance)
}

// handleServiceBackoffLimit checks if service has BackoffLimit and translate it to a warning duration with back-off limit
func (r *ServiceReconciler) handleServiceBackoffLimit(ctx context.Context, instance *apiv1.Service) error {
	log := r.Log

	backoffLimit := instance.Termination.BackoffLimit
	if backoffLimit == nil {
		return nil
	}
	lastTransitionTime := instance.Status.Conditions[len(instance.Status.Conditions)-1].LastTransitionTime
	currentTime := metav1.Now()
	duration := currentTime.Time.Sub(lastTransitionTime.Time)

	if duration >= r.getBackOff(*backoffLimit) {
		log.V(1).Info("Cleanup triggered based on ActiveDeadlineSeconds")
		return r.delete(ctx, instance)
	}

	return nil
}

// handleCulling checks if the service is idle and should be culled
func (r *ServiceReconciler) handleCulling(ctx context.Context, instance *apiv1.Service) (*time.Duration, error) {
	log := r.Log

	if instance.Termination.Culling == nil || instance.Termination.Culling.Timeout == nil {
		return nil, nil
	}

	// Log the culling timeout
	log.V(1).Info("Culling timeout", "timeout", instance.Termination.Culling.Timeout)

	// Check if the service is running
	if !instance.Status.IsRunning() {
		return nil, nil
	}

	timeout := time.Duration(*instance.Termination.Culling.Timeout) * time.Second

	// Check for activity
	var lastActivity time.Time
	if instance.Termination.Probe != nil && instance.Termination.Probe.Http != nil {
		var err error
		lastActivity, err = r.checkHttpActivity(ctx, instance, instance.Termination.Probe.Http)
		if err != nil {
			log.Error(err, "Failed to check http activity")
			// Assume active on error to avoid accidental culling
			// Requeue to check again later
			return &timeout, nil
		}
	} else if instance.Termination.Probe != nil && instance.Termination.Probe.Exec != nil {
		// Exec probe is defined in the API but not yet implemented
		log.Info("Exec probe configured but not yet implemented, skipping culling check")
		return nil, nil
	} else {
		// No probe configured, cannot cull based on activity
		log.Info("No probe configured, skipping culling check")
		return nil, nil
	}

	elapsed := time.Since(lastActivity)

	if elapsed < timeout {
		// Not idle long enough
		// Requeue for the remaining time
		remaining := timeout - elapsed
		return &remaining, nil
	}

	log.Info("Service is idle, triggering culling", "idleDuration", elapsed, "timeout", timeout)
	return nil, r.delete(ctx, instance)
}

type HttpStatus struct {
	LastActivity string `json:"last_activity"`
	Started      string `json:"started"`
}

func (r *ServiceReconciler) checkHttpActivity(ctx context.Context, instance *apiv1.Service, probe *apiv1.ActivityProbeHttp) (time.Time, error) {
	log := r.Log

	// First verify the k8s Service exists
	foundService := &corev1.Service{}
	serviceName := instance.Name
	if instance.ServiceSpec != nil && instance.ServiceSpec.IsExternal {
		serviceName += "-ext"
	}

	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: instance.Namespace}, foundService)
	if err != nil {
		if apierrs.IsNotFound(err) {
			log.V(1).Info("K8s Service not found yet, will retry later", "serviceName", serviceName, "namespace", instance.Namespace)
			return time.Time{}, fmt.Errorf("service %s not found in namespace %s", serviceName, instance.Namespace)
		}
		return time.Time{}, err
	}

	log.V(1).Info("K8s Service found", "serviceName", serviceName, "namespace", instance.Namespace, "clusterIP", foundService.Spec.ClusterIP)

	// Determine the port to use
	var port int32
	if probe.Port != 0 {
		// Use explicitly configured port
		port = probe.Port
	} else {
		// Default to first service port
		ports := managers.GetPodPorts(instance.ServiceSpec.Template.Spec, managers.DefaultTargetPort)
		if instance.ServiceSpec.Ports != nil && len(instance.ServiceSpec.Ports) > 0 {
			ports = instance.ServiceSpec.Ports
		}

		if len(ports) == 0 {
			return time.Time{}, fmt.Errorf("no ports defined for service")
		}
		port = ports[0]
	}

	// Handle path - use configured path or default to /api/status
	path := probe.Path
	if path == "" {
		path = "/api/status"
	}
	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Access service directly - Jupyter is configured with base_url matching the proxy path
	host := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, instance.Namespace)

	// Get instance UUID from label
	instanceID := ""
	if instance.ObjectMeta.Labels != nil {
		instanceID = instance.ObjectMeta.Labels["app.kubernetes.io/instance"]
	}
	if instanceID == "" {
		return time.Time{}, fmt.Errorf("missing required label app.kubernetes.io/instance")
	}

	// Get owner and project from annotations
	owner := ""
	project := ""
	if instance.Annotations != nil {
		owner = instance.Annotations["operation.polyaxon.com/owner"]
		project = instance.Annotations["operation.polyaxon.com/project"]
	}
	if owner == "" || project == "" {
		return time.Time{}, fmt.Errorf("missing required annotations operation.polyaxon.com/owner or operation.polyaxon.com/project")
	}

	// Construct full Polyaxon service path: /services/v1/{namespace}/{owner}/{project}/runs/{uuid}/{port}{path}
	fullPath := fmt.Sprintf("/services/v1/%s/%s/%s/runs/%s/%d%s",
		instance.Namespace, owner, project, instanceID, port, path)

	url := fmt.Sprintf("http://%s:%d%s", host, port, fullPath)

	log.V(1).Info("Checking HTTP activity", "url", url)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return time.Time{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "HTTP request failed", "url", url, "timeout", client.Timeout)
		return time.Time{}, err
	}
	defer resp.Body.Close()

	log.V(1).Info("HTTP response received", "status", resp.StatusCode, "url", url)

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("http api returned status: %d", resp.StatusCode)
	}

	var status HttpStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return time.Time{}, err
	}

	lastActivity, err := time.Parse(time.RFC3339, status.LastActivity)
	if err != nil {
		return time.Time{}, err
	}

	return lastActivity, nil
}
