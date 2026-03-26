package tui

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "00:00"},
		{"seconds only", 45 * time.Second, "00:45"},
		{"one minute", 60 * time.Second, "01:00"},
		{"mixed", 3*time.Minute + 7*time.Second, "03:07"},
		{"rounds sub-second", 2*time.Minute + 30*time.Second + 600*time.Millisecond, "02:31"},
		{"large", 15*time.Minute + 59*time.Second, "15:59"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestStatusLine(t *testing.T) {
	healthy := StatusLine("Flux CD", true, "v2.6.4")
	if healthy == "" {
		t.Fatal("expected non-empty StatusLine for healthy")
	}
	unhealthy := StatusLine("Gateway", false, "pending")
	if unhealthy == "" {
		t.Fatal("expected non-empty StatusLine for unhealthy")
	}
}

func TestHeader(t *testing.T) {
	h := Header("Test Section")
	if h == "" {
		t.Fatal("expected non-empty Header")
	}
}

func TestVerboseLines(t *testing.T) {
	got := VerboseLines(false, []string{"detail"})
	if got != "" {
		t.Errorf("VerboseLines(false, ...) = %q, want empty", got)
	}

	got = VerboseLines(true, nil)
	if got != "" {
		t.Errorf("VerboseLines(true, nil) = %q, want empty", got)
	}

	got = VerboseLines(true, []string{"line1", "line2"})
	if got == "" {
		t.Fatal("expected non-empty VerboseLines")
	}
}
