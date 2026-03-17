package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/log"
)

const DefaultClusterName = "shoulders"

func EnsureKindCluster(name string, kindConfig []byte) (err error) {
	provider, err := newProvider()
	if err != nil {
		return err
	}
	clusters, err := provider.List()
	if err != nil {
		return err
	}
	for _, existing := range clusters {
		if existing == name {
			return nil
		}
	}

	// Write the embedded config to a temp file for the kind library
	tmpDir, err := os.MkdirTemp("", "shoulders-kind-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove temp dir %q: %w", tmpDir, removeErr))
		}
	}()

	configPath := filepath.Join(tmpDir, "kind-config.yaml")
	if err := os.WriteFile(configPath, kindConfig, 0o644); err != nil {
		return fmt.Errorf("write kind config: %w", err)
	}

	return provider.Create(
		name,
		cluster.CreateWithConfigFile(configPath),
		cluster.CreateWithWaitForReady(5*time.Minute),
	)
}

func DeleteKindCluster(name string) error {
	provider, err := newProvider()
	if err != nil {
		return err
	}
	return provider.Delete(name, "")
}

func ListClusters() ([]string, error) {
	provider, err := newProvider()
	if err != nil {
		return nil, err
	}
	return provider.List()
}

func newProvider() (*cluster.Provider, error) {
	opt, err := cluster.DetectNodeProvider()
	if err != nil {
		return nil, fmt.Errorf("no container runtime detected; please install Docker or Podman: %w", err)
	}
	return cluster.NewProvider(opt, cluster.ProviderWithLogger(log.NoopLogger{})), nil
}
