package cmd

import (
	"testing"

	"github.com/jherreros/shoulders/shoulders-cli/internal/config"
)

func TestCurrentNamespaceOverride(t *testing.T) {
	originalOverride := namespaceOverride
	originalConfig := currentConfig
	defer func() {
		namespaceOverride = originalOverride
		currentConfig = originalConfig
	}()

	namespaceOverride = "override"
	currentConfig = &config.Config{CurrentWorkspace: "team-a"}

	ns, err := currentNamespace()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ns != "override" {
		t.Fatalf("expected override namespace, got %s", ns)
	}
}

func TestCurrentNamespaceFromConfig(t *testing.T) {
	originalOverride := namespaceOverride
	originalConfig := currentConfig
	defer func() {
		namespaceOverride = originalOverride
		currentConfig = originalConfig
	}()

	namespaceOverride = ""
	currentConfig = &config.Config{CurrentWorkspace: "team-b"}

	ns, err := currentNamespace()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ns != "team-b" {
		t.Fatalf("expected team-b namespace, got %s", ns)
	}
}

func TestCurrentNamespaceMissing(t *testing.T) {
	originalOverride := namespaceOverride
	originalConfig := currentConfig
	defer func() {
		namespaceOverride = originalOverride
		currentConfig = originalConfig
	}()

	namespaceOverride = ""
	currentConfig = nil

	_, err := currentNamespace()
	if err == nil {
		t.Fatalf("expected error when namespace missing")
	}
}

func TestGatewayChecksRequiredDefaultsToTrue(t *testing.T) {
	originalConfig := currentConfig
	defer func() {
		currentConfig = originalConfig
	}()

	currentConfig = nil
	if !gatewayChecksRequired() {
		t.Fatalf("expected gateway checks to be required by default")
	}
}

func TestGatewayChecksRequiredDisabledWhenCiliumDisabled(t *testing.T) {
	originalConfig := currentConfig
	defer func() {
		currentConfig = originalConfig
	}()

	enabled := false
	currentConfig = &config.Config{Platform: config.PlatformConfig{Cilium: config.CiliumConfig{Enabled: &enabled}}}
	if gatewayChecksRequired() {
		t.Fatalf("expected gateway checks to be skipped when cilium is disabled")
	}
}

func TestOnlyDeferredFluxKustomizations(t *testing.T) {
	tests := []struct {
		name    string
		pending []string
		want    bool
	}{
		{name: "Empty", pending: nil, want: false},
		{name: "AllDeferred", pending: []string{"helm-releases", "gateway"}, want: true},
		{name: "ContainsNonDeferred", pending: []string{"helm-releases", "namespaces"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := onlyDeferredFluxKustomizations(tt.pending); got != tt.want {
				t.Fatalf("onlyDeferredFluxKustomizations() = %v, want %v", got, tt.want)
			}
		})
	}
}
