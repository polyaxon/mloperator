package service

import (
	"time"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/helpers/config"
	"github.com/polyaxon/mloperator/internal/helpers/plugins"
)

const (
	apiServerDefaultTimeout = 35 * time.Second
)

func (r *ServiceReconciler) instanceSyncStatus(instance *apiv1.Service) error {
	lastCond := instance.Status.Conditions[len(instance.Status.Conditions)-1]
	return r.syncStatus(instance, lastCond)
}

func (r *ServiceReconciler) getInstanceInfo(instance *apiv1.Service) (string, string, string, string, bool) {
	instanceID, ok := instance.ObjectMeta.Labels["app.kubernetes.io/instance"]
	if !ok || instanceID == "" {
		return "", "", "", "", false
	}

	instanceOwner, ok := instance.ObjectMeta.Annotations["operation.polyaxon.com/owner"]
	if !ok || instanceOwner == "" {
		return "", "", "", "", false
	}

	instanceProject, ok := instance.ObjectMeta.Annotations["operation.polyaxon.com/project"]
	if !ok || instanceProject == "" {
		return "", "", "", "", false
	}

	instanceKind, ok := instance.ObjectMeta.Annotations["operation.polyaxon.com/kind"]
	if !ok || instanceKind == "" {
		instanceKind = "operation" // backward compatibility
	}

	return instanceOwner, instanceProject, instanceID, instanceKind, true
}

func (r *ServiceReconciler) syncStatus(instance *apiv1.Service, statusCond apiv1.OperationCondition) error {
	if !config.GetBoolEnv(config.AgentEnabled, true) || !instance.SyncStatuses {
		return nil
	}

	log := r.Log

	log.Info("Operation sync status", "Syncing", instance.GetName(), "Status", statusCond.Type)
	owner, project, instanceID, _, ok := r.getInstanceInfo(instance)
	if !ok {
		log.Info("Operation cannot be synced", "Instance", instance.Name, "Uuid Does not exist", instance.GetName())
		return nil
	}
	return plugins.LogPolyaxonRunStatus(owner, project, instanceID, statusCond, r.Log)
}

func (r *ServiceReconciler) collectLogs(instance *apiv1.Service) error {

	if !config.GetBoolEnv(config.AgentEnabled, true) || !instance.CollectLogs {
		return nil
	}

	log := r.Log

	owner, project, instanceID, runKind, ok := r.getInstanceInfo(instance)
	if !ok {
		log.Info("Operation cannot be synced", "Instance", instance.Name, "Uuid Does not exist", instance.GetName())
		return nil
	}

	log.Info("Operation collect logs", "Instance", instance.GetName(), "kind", runKind)
	return plugins.CollectPolyaxonRunLogs(instance.Namespace, owner, project, instanceID, runKind, r.Log)
}
