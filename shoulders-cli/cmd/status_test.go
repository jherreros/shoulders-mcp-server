package cmd

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIsPodHealthy(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1.Pod
		want bool
	}{
		{
			name: "RunningAndReady",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionTrue},
					},
				},
			},
			want: true,
		},
		{
			name: "RunningButNotReady",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionFalse},
					},
				},
			},
			want: false,
		},
		{
			name: "RunningNoConditions",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			want: false,
		},
		{
			name: "Succeeded",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
			want: true,
		},
		{
			name: "Pending",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			want: false,
		},
		{
			name: "Failed",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPodHealthy(tt.pod); got != tt.want {
				t.Errorf("isPodHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name   string
		ready  bool
		issues []string
		want   string
	}{
		{"Ready", true, nil, "Healthy"},
		{"ReadyWithIssues", true, []string{"warn"}, "Healthy"}, // Should probably not happen logic-wise but good to assert behavior
		{"NotReadyNoIssues", false, nil, "Unhealthy"},
		{"NotReadyWithIssues", false, []string{"err1"}, "Unhealthy (err1)"},
		{"NotReadyMultipleIssues", false, []string{"err1", "err2"}, "Unhealthy (err1, err2)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatStatus(tt.ready, tt.issues); got != tt.want {
				t.Errorf("formatStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoolToText(t *testing.T) {
	if got := boolToText(true); got != "Ready" {
		t.Errorf("boolToText(true) = %v, want Ready", got)
	}
	if got := boolToText(false); got != "Not Ready" {
		t.Errorf("boolToText(false) = %v, want Not Ready", got)
	}
}
