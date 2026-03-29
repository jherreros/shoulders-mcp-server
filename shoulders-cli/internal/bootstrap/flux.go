package bootstrap

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
)

const fluxInstallURL = "https://github.com/fluxcd/flux2/releases/download/v2.8.3/install.yaml"

func EnsureFlux(ctx context.Context, kubeconfigPath string, fluxManifests ...[]byte) error {
	manifest, err := downloadFluxManifest(ctx)
	if err != nil {
		return err
	}
	if err := kube.ApplyManifest(ctx, kubeconfigPath, manifest, ""); err != nil {
		return fmt.Errorf("apply flux install manifest: %w", err)
	}

	for _, content := range fluxManifests {
		if err := kube.ApplyManifest(ctx, kubeconfigPath, content, "flux-system"); err != nil {
			return fmt.Errorf("apply flux config: %w", err)
		}
	}
	return nil
}

func downloadFluxManifest(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fluxInstallURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to download flux install manifest: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
