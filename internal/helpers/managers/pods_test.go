package managers

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
)

func newPodsClient(t *testing.T, pods ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(pods...).Build()
}

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

			podStatus, reason, message := HasUnschedulablePods(newPodsClient(t, pod), "op", "default")

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
