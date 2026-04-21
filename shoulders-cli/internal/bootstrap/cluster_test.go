package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUseAnonymousDockerConfigIfNeededCreatesTempConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DOCKER_CONFIG", "")

	dockerDir := filepath.Join(home, ".docker")
	if err := os.MkdirAll(dockerDir, 0o755); err != nil {
		t.Fatalf("mkdir docker dir: %v", err)
	}

	config := map[string]any{
		"auths": map[string]map[string]string{
			"localhost:5006": {},
		},
		"credsStore": "desktop",
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cleanup, err := useAnonymousDockerConfigIfNeeded()
	if err != nil {
		t.Fatalf("useAnonymousDockerConfigIfNeeded() error = %v", err)
	}
	defer cleanup()

	tmpDir := os.Getenv("DOCKER_CONFIG")
	if tmpDir == "" {
		t.Fatal("expected DOCKER_CONFIG to be set")
	}
	if tmpDir == dockerDir {
		t.Fatal("expected DOCKER_CONFIG to point to a temporary directory")
	}

	tmpData, err := os.ReadFile(filepath.Join(tmpDir, "config.json"))
	if err != nil {
		t.Fatalf("read temp config: %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(tmpData, &got); err != nil {
		t.Fatalf("unmarshal temp config: %v", err)
	}
	if _, ok := got["credsStore"]; ok {
		t.Fatal("expected temp docker config to omit credsStore")
	}
	if _, ok := got["auths"]; !ok {
		t.Fatal("expected temp docker config to preserve auths")
	}
}

func TestUseAnonymousDockerConfigIfNeededRespectsExistingOverride(t *testing.T) {
	override := t.TempDir()
	t.Setenv("DOCKER_CONFIG", override)

	cleanup, err := useAnonymousDockerConfigIfNeeded()
	if err != nil {
		t.Fatalf("useAnonymousDockerConfigIfNeeded() error = %v", err)
	}
	defer cleanup()

	if got := os.Getenv("DOCKER_CONFIG"); got != override {
		t.Fatalf("expected DOCKER_CONFIG to stay %q, got %q", override, got)
	}
}