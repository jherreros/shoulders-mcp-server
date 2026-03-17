package bootstrap

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAPIServerPodObservedStartTimePrefersRunningContainerStart(t *testing.T) {
	podStart := metav1.NewTime(time.Date(2026, time.March, 15, 20, 6, 3, 0, time.UTC))
	containerStart := metav1.NewTime(time.Date(2026, time.March, 15, 20, 17, 12, 0, time.UTC))

	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			StartTime: &podStart,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "kube-apiserver",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{StartedAt: containerStart},
				},
			}},
		},
	}

	got := apiserverPodObservedStartTime(pod)
	if got != containerStart.UTC().Format(time.RFC3339) {
		t.Fatalf("expected running container start time %q, got %q", containerStart.UTC().Format(time.RFC3339), got)
	}
}

func TestAPIServerPodObservedStartTimeFallsBackToPodStart(t *testing.T) {
	podStart := metav1.NewTime(time.Date(2026, time.March, 15, 20, 6, 3, 0, time.UTC))

	pod := &corev1.Pod{
		Status: corev1.PodStatus{StartTime: &podStart},
	}

	got := apiserverPodObservedStartTime(pod)
	if got != podStart.UTC().Format(time.RFC3339) {
		t.Fatalf("expected pod start time %q, got %q", podStart.UTC().Format(time.RFC3339), got)
	}
}
