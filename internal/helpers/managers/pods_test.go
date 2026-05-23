package managers

import (
	"strings"
	"testing"
)

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
