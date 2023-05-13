package controllers

import (
	"time"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/controllers/config"
	"github.com/polyaxon/mloperator/controllers/plugins"
)

const (
	apiServerDefaultTimeout = 35 * time.Second
)

func (r *OperationReconciler) instanceSyncStatus(instance *operationv1.Operation) error {
	lastCond := instance.Status.Conditions[len(instance.Status.Conditions)-1]
	return r.syncStatus(instance, lastCond)
}

func (r *OperationReconciler) getInstanceInfo(instance *operationv1.Operation) (string, string, string, string, bool) {
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

func (r *OperationReconciler) syncStatus(instance *operationv1.Operation, statusCond operationv1.OperationCondition) error {
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

func (r *OperationReconciler) notify(instance *operationv1.Operation) error {

	if !config.GetBoolEnv(config.AgentEnabled, true) || len(instance.Notifications) == 0 {
		return nil
	}

	log := r.Log

	log.Info("Operation notify status", "Notifying", instance.GetName())

	owner, project, instanceID, _, ok := r.getInstanceInfo(instance)
	if !ok {
		log.Info("Operation cannot be synced", "Instance", instance.Name, "Uuid Does not exist", instance.GetName())
		return nil
	}

	name, ok := instance.ObjectMeta.Annotations["operation.polyaxon.com/name"]
	if !ok {
		name = ""
	}

	if len(instance.Status.Conditions) == 0 {
		log.Info("Operation cannot be notified", "Instance", instance.Name, "No conditions", instance.GetName())
		return nil
	}
	lastCond := instance.Status.Conditions[len(instance.Status.Conditions)-1]

	connections := []string{}
	for _, notification := range instance.Notifications {
		if notification.Trigger == operationv1.OperationDoneTrigger || operationv1.OperationConditionType(notification.Trigger) == lastCond.Type {
			connections = append(connections, notification.Connections...)
		}
	}

	if len(connections) == 0 {
		log.Info("Operation no notification", "Instance", instance.Name, "No connections for status", lastCond.Type)
		return nil
	}

	log.Info("Operation notify status", "Status", lastCond.Type, "Instance", instance.GetName())
	return plugins.NotifyPolyaxonRunStatus(instance.Namespace, name, owner, project, instanceID, lastCond, connections, r.Log)
}

func (r *OperationReconciler) collectLogs(instance *operationv1.Operation) error {

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
