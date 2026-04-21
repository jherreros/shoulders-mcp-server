package config

import (
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestConfigSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	enabled := false
	cfg := &Config{
		CurrentWorkspace: "team-a",
		Cluster: ClusterConfig{
			Provider: ProviderExisting,
			Context:  "demo-context",
		},
		Platform: PlatformConfig{
			Cilium: CiliumConfig{Enabled: &enabled},
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.CurrentWorkspace != "team-a" {
		t.Fatalf("expected workspace team-a, got %s", loaded.CurrentWorkspace)
	}
	if loaded.Provider() != ProviderExisting {
		t.Fatalf("expected existing provider, got %s", loaded.Provider())
	}
	if loaded.Cluster.Context != "demo-context" {
		t.Fatalf("expected demo-context, got %s", loaded.Cluster.Context)
	}
	if loaded.CiliumEnabled() {
		t.Fatalf("expected cilium disabled after load")
	}
	if loaded.FluxPathPrefix() != "." {
		t.Fatalf("expected default flux path prefix '.', got %s", loaded.FluxPathPrefix())
	}

	configPath, err := Path()
	if err != nil {
		t.Fatalf("path failed: %v", err)
	}
	expected := filepath.Join(tmpDir, ".shoulders", "config.yaml")
	if configPath != expected {
		t.Fatalf("expected config path %s, got %s", expected, configPath)
	}
}

func TestLoadDefaultsWithoutConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Provider() != ProviderVind {
		t.Fatalf("expected vind provider, got %s", loaded.Provider())
	}
	if loaded.ClusterName() != DefaultClusterName {
		t.Fatalf("expected default cluster name %s, got %s", DefaultClusterName, loaded.ClusterName())
	}
	if !loaded.CiliumEnabled() {
		t.Fatalf("expected cilium enabled by default")
	}
	if loaded.DexHost() != DefaultDexHost {
		t.Fatalf("expected default dex host %s, got %s", DefaultDexHost, loaded.DexHost())
	}
}

func TestLoadAppliesProviderSpecificDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "shoulders.yaml")
	content := []byte("cluster:\n  provider: existing\n")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Provider() != ProviderExisting {
		t.Fatalf("expected existing provider, got %s", loaded.Provider())
	}
	if loaded.CiliumEnabled() {
		t.Fatalf("expected cilium disabled for existing cluster defaults")
	}
}

func TestCustomDomainHosts(t *testing.T) {
	cfg := &Config{Platform: PlatformConfig{Domain: "lvh.me"}}
	cfg.ApplyDefaults()

	if cfg.DexHost() != "dex.lvh.me" {
		t.Fatalf("expected custom dex host, got %s", cfg.DexHost())
	}
	if cfg.GrafanaHost() != "grafana.lvh.me" {
		t.Fatalf("expected custom grafana host, got %s", cfg.GrafanaHost())
	}
	if cfg.HeadlampHost() != "headlamp.lvh.me" {
		t.Fatalf("expected custom headlamp host, got %s", cfg.HeadlampHost())
	}
}

func TestExampleYAMLIsValid(t *testing.T) {
	content, err := ExampleYAML(ProviderExisting)
	if err != nil {
		t.Fatalf("example generation failed: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		t.Fatalf("example YAML should parse: %v", err)
	}
	if cfg.Cluster.Provider != ProviderExisting {
		t.Fatalf("expected existing provider in example, got %s", cfg.Cluster.Provider)
	}
	if cfg.FluxPathPrefix() != "." {
		t.Fatalf("expected default path prefix '.', got %s", cfg.FluxPathPrefix())
	}
}
