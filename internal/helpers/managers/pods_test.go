package managers

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
)

func newPendingPod(name string, instanceID string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    map[string]string{"app.kubernetes.io/instance": instanceID},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{
					Type:    corev1.PodReady,
					Status:  corev1.ConditionFalse,
					Reason:  "ContainersNotReady",
					Message: "containers with unready status: [polyaxon-main]",
				},
			},
		},
	}
}

func waitingStatus(reason string, message string) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name: "polyaxon-main",
		State: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message},
		},
	}
}

func TestHasUnschedulablePodsReportsErrImagePullAsImagePullBackOff(t *testing.T) {
	for _, tc := range []struct {
		name   string
		init   bool
		reason string
	}{
		{name: "main container", reason: "ImagePullBackOff"},
		{name: "init container", init: true, reason: "InitContainerImagePullBackOff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := newPendingPod("op-pod", "op")
			statuses := []corev1.ContainerStatus{waitingStatus("ErrImagePull", "pull access denied")}
			if tc.init {
				pod.Status.InitContainerStatuses = statuses
			} else {
				pod.Status.ContainerStatuses = statuses
			}

			podStatus, reason, message := HasUnschedulablePods([]corev1.Pod{*pod})

			if podStatus != apiv1.OperationWarning || reason != tc.reason || message != "pull access denied" {
				t.Fatalf("got %q %q %q, want Warning %q %q", podStatus, reason, message, tc.reason, "pull access denied")
			}
		})
	}
}

func TestFormatMainContainerFailureMessage(t *testing.T) {
	fallback := "Job has reached the specified backoff limit"
	if got := FormatMainContainerFailureMessage(fallback, nil, nil); got != fallback {
		t.Fatalf("nil exit status message = %q, want %q", got, fallback)
	}

	attempts := int32(3)
	got := FormatMainContainerFailureMessage(fallback, &MainContainerExitStatus{
		PodName:       "run-abc-xyz",
		ContainerName: "trainer",
		ExitCode:      2,
		Reason:        "Error",
		Message:       "bad args",
	}, &attempts)

	for _, want := range []string{
		"after 3 failed attempt(s)",
		`Main container "trainer" in pod "run-abc-xyz"`,
		"exit code 2 (Error): bad args",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("message = %q, want substring %q", got, want)
		}
	}
}

func TestGetInvalidImageName(t *testing.T) {
	for _, tc := range []struct {
		name   string
		init   bool
		reason string
		found  bool
	}{
		{name: "main container", reason: InvalidImageNameReason, found: true},
		{name: "init container", init: true, reason: InvalidImageNameReason, found: true},
		{name: "image pull retries", reason: imagePullBackOffReason},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := newPendingPod("op-pod", "op")
			statuses := []corev1.ContainerStatus{waitingStatus(tc.reason, "bad image")}
			if tc.init {
				pod.Status.InitContainerStatuses = statuses
			} else {
				pod.Status.ContainerStatuses = statuses
			}

			message, found := GetInvalidImageName([]corev1.Pod{*pod})

			if found != tc.found || (found && message != "bad image") {
				t.Fatalf("got %q %v, want found=%v", message, found, tc.found)
			}
		})
	}

	t.Run("no pods", func(t *testing.T) {
		if _, found := GetInvalidImageName(nil); found {
			t.Fatal("found an invalid image name without pods")
		}
	})
}

func TestIsMainContainerRunning(t *testing.T) {
	running := func(name string) corev1.ContainerStatus {
		return corev1.ContainerStatus{
			Name:  name,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
		}
	}
	for _, tc := range []struct {
		name     string
		statuses []corev1.ContainerStatus
		want     bool
	}{
		{name: "main running", statuses: []corev1.ContainerStatus{running("polyaxon-main")}, want: true},
		{name: "main waiting", statuses: []corev1.ContainerStatus{waitingStatus("ContainerCreating", "")}},
		{
			name:     "sidecar running before main",
			statuses: []corev1.ContainerStatus{running("polyaxon-sidecar"), waitingStatus("ContainerCreating", "")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := newPendingPod("op-pod", "op")
			pod.Spec.Containers = []corev1.Container{{Name: "polyaxon-main"}, {Name: "polyaxon-sidecar"}}
			pod.Status.ContainerStatuses = tc.statuses

			if got := IsMainContainerRunning([]corev1.Pod{*pod}); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("no pods", func(t *testing.T) {
		if IsMainContainerRunning(nil) {
			t.Fatal("main container running without pods")
		}
	})
}
