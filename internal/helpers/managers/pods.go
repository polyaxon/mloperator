package managers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MainContainerExitStatus struct {
	PodName         string
	ContainerName   string
	ExitCode        int32
	Reason          string
	Message         string
	RestartCount    int32
	currentExitSeen bool
}

func (status MainContainerExitStatus) ContainerFailedAttempts() int32 {
	attempts := status.RestartCount
	if status.currentExitSeen {
		attempts++
	}
	return attempts
}

// GetPodPorts returns the pod's port from the container definition
func GetPodPorts(podSpec corev1.PodSpec, defaultPort int) []int32 {
	ports := []int32{int32(defaultPort)}
	containerPorts := podSpec.Containers[0].Ports
	if containerPorts != nil {
		ports = []int32{}
		for _, cp := range containerPorts {
			ports = append(ports, cp.ContainerPort)
		}
	}
	return ports
}

func getPodLastTime(pod *corev1.Pod, lastTime *time.Time) (bool, *time.Time) {
	timeRaw := pod.CreationTimestamp.Time
	if lastTime == nil || lastTime.Before(timeRaw) {
		return true, &timeRaw
	}

	return false, lastTime
}

// GetLastPod returns the last pod bassed on the creation time of the items
func GetLastPod(pods corev1.PodList) (*corev1.Pod, error) {
	lastTime := &time.Time{}
	lastPod := &corev1.Pod{}
	isLast := false
	var err error
	for _, pod := range pods.Items {
		isLast, lastTime = getPodLastTime(&pod, lastTime)
		if err != nil {
			return nil, err
		}
		if isLast {
			lastPod = &pod
		}
	}
	return lastPod, nil
}

// ListPods returns the list of pods based on selctor
func ListPods(controllerClient client.Client, namespace string, selector map[string]string) (*corev1.PodList, error) {

	clientOpt := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(selector),
	}

	opt := []client.ListOption{
		clientOpt,
	}

	podList := &corev1.PodList{}
	return podList, controllerClient.List(context.TODO(), podList, opt...)
}

func getPodMainContainerExitStatus(pod corev1.Pod) *MainContainerExitStatus {
	if len(pod.Status.ContainerStatuses) == 0 {
		return nil
	}

	containerName := pod.Status.ContainerStatuses[0].Name
	if len(pod.Spec.Containers) > 0 {
		containerName = pod.Spec.Containers[0].Name
	}

	var containerStatus *corev1.ContainerStatus
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == containerName {
			containerStatus = &pod.Status.ContainerStatuses[i]
			break
		}
	}
	if containerStatus == nil {
		return nil
	}

	currentExitSeen := false
	var termination *corev1.ContainerStateTerminated
	if containerStatus.State.Terminated != nil {
		termination = containerStatus.State.Terminated
		currentExitSeen = true
	} else if containerStatus.State.Running != nil {
		lastTermination := containerStatus.LastTerminationState.Terminated
		if lastTermination != nil && lastTermination.ExitCode != 0 {
			termination = lastTermination
		}
	} else if containerStatus.State.Waiting != nil {
		// Waiting with a previous clean exit is unusual, but means the pod is not healthy now.
		// Keep honoring it so single-container command services can complete cleanly.
		termination = containerStatus.LastTerminationState.Terminated
	}
	if termination == nil {
		return nil
	}

	return &MainContainerExitStatus{
		PodName:         pod.Name,
		ContainerName:   containerName,
		ExitCode:        termination.ExitCode,
		Reason:          termination.Reason,
		Message:         termination.Message,
		RestartCount:    containerStatus.RestartCount,
		currentExitSeen: currentExitSeen,
	}
}

func GetMainContainerExitStatus(pods corev1.PodList) *MainContainerExitStatus {
	var failedStatus *MainContainerExitStatus
	var succeededStatus *MainContainerExitStatus
	for _, pod := range pods.Items {
		status := getPodMainContainerExitStatus(pod)
		if status == nil {
			continue
		}
		if status.ExitCode == 0 {
			if succeededStatus == nil {
				succeededStatus = status
			}
			continue
		}
		if failedStatus == nil || status.ContainerFailedAttempts() > failedStatus.ContainerFailedAttempts() {
			failedStatus = status
		}
	}
	if failedStatus != nil {
		return failedStatus
	}
	return succeededStatus
}

func GetMainContainerExitStatusByInstance(controllerClient client.Client, instanceID string, namespace string) (*MainContainerExitStatus, error) {
	selector := map[string]string{
		"app.kubernetes.io/instance": instanceID,
	}
	podsList, err := ListPods(controllerClient, namespace, selector)
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	if len(podsList.Items) == 0 {
		return nil, nil
	}
	return GetMainContainerExitStatus(*podsList), nil
}

// HasUnschedulablePods Detects if entity has unschedulable pods
func HasUnschedulablePods(controllerClient client.Client, instanceID string, namespace string) (apiv1.OperationConditionType, string, string) {
	_labels := map[string]string{
		"app.kubernetes.io/instance": instanceID,
	}
	podsList, err := ListPods(controllerClient, namespace, _labels)
	if err != nil || len(podsList.Items) < 1 {
		return apiv1.OperationStarting, "PodNotReady", "Operation has no pods yet."
	}
	for _, pod := range podsList.Items {
		if pod.Status.Phase == corev1.PodFailed {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Terminated != nil && cs.State.Terminated.ExitCode > 0 {
					return apiv1.OperationFailed, cs.State.Terminated.Reason + "ExitCode " + strconv.Itoa(int(cs.State.Terminated.ExitCode)), cs.State.Terminated.Message
				}
			}
			return apiv1.OperationFailed, "PodFailed", pod.Status.Message
		}
		if pod.Status.Phase == corev1.PodSucceeded {
			return "", "", ""
		}
		if pod.Status.Phase != corev1.PodRunning && pod.Status.Conditions != nil {
			if pod.Status.InitContainerStatuses != nil {
				for _, cs := range pod.Status.InitContainerStatuses {
					if !cs.Ready && cs.State.Waiting != nil && cs.State.Waiting.Reason == "ImagePullBackOff" {
						return apiv1.OperationWarning, "InitContainerImagePullBackOff", cs.State.Waiting.Message
					}
				}
			}
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready && cs.State.Waiting != nil && cs.State.Waiting.Reason == "ImagePullBackOff" {
					return apiv1.OperationWarning, "ImagePullBackOff", cs.State.Waiting.Message
				}
			}
			for _, cond := range pod.Status.Conditions {
				if (cond.Reason == corev1.PodReasonUnschedulable) ||
					(cond.Type == corev1.PodReady && cond.Status == corev1.ConditionFalse && cond.Reason == "ContainersNotReady") {
					return apiv1.OperationWarning, "ContainersNotReady", cond.Message
				}
			}
		}
		if pod.Status.Phase == corev1.PodReasonUnschedulable {
			return apiv1.OperationWarning, "PodReasonUnschedulable", "Pod is unschedulable"
		}
		if pod.Status.Phase == corev1.PodPending && pod.Status.Conditions == nil {
			return apiv1.OperationWarning, "PodPending", "Pod is still pending without conditions"
		}
	}
	return "", "", ""
}
