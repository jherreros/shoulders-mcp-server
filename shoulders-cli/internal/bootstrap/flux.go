package bootstrap

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

const fluxInstallURL = "https://github.com/fluxcd/flux2/releases/download/v2.8.3/install.yaml"

const fluxPlatformConfigName = "shoulders-platform-config"

var fluxKustomizationGVR = schema.GroupVersionResource{
	Group:    "kustomize.toolkit.fluxcd.io",
	Version:  "v1",
	Resource: "kustomizations",
}

func EnsureFlux(ctx context.Context, kubeconfigPath, repoURL, branch, pathPrefix string, publicConfig PublicDomainConfig) error {
	manifest, err := downloadFluxManifest(ctx)
	if err != nil {
		return err
	}
	if err := kube.ApplyManifest(ctx, kubeconfigPath, manifest, ""); err != nil {
		return fmt.Errorf("apply flux install manifest: %w", err)
	}
	if err := kube.ApplyManifest(ctx, kubeconfigPath, fluxPlatformConfigManifest(publicConfig), "flux-system"); err != nil {
		return fmt.Errorf("apply flux platform config: %w", err)
	}
	if err := kube.ApplyManifest(ctx, kubeconfigPath, fluxGitRepositoryManifest(repoURL, branch), "flux-system"); err != nil {
		return fmt.Errorf("apply flux git repository: %w", err)
	}
	if err := kube.ApplyManifest(ctx, kubeconfigPath, fluxKustomizationsManifest(pathPrefix), "flux-system"); err != nil {
		return fmt.Errorf("apply flux config: %w", err)
	}
	return nil
}

func UninstallShouldersFlux(ctx context.Context, kubeconfigPath, pathPrefix string) error {
	installed, err := fluxAPIsInstalled(kubeconfigPath)
	if err != nil {
		return err
	}
	if !installed {
		return nil
	}

	if err := kube.DeleteManifest(ctx, kubeconfigPath, fluxKustomizationsManifest(pathPrefix), "flux-system"); err != nil {
		return fmt.Errorf("delete flux kustomizations: %w", err)
	}
	if err := waitForFluxKustomizationsDeleted(ctx, kubeconfigPath); err != nil {
		return err
	}
	if err := kube.DeleteManifest(ctx, kubeconfigPath, fluxPlatformConfigManifest(PublicDomainConfig{}), "flux-system"); err != nil {
		return fmt.Errorf("delete flux platform config: %w", err)
	}
	if err := kube.DeleteManifest(ctx, kubeconfigPath, fluxGitRepositoryManifest("", ""), "flux-system"); err != nil {
		return fmt.Errorf("delete flux git repository: %w", err)
	}
	return nil
}

func fluxAPIsInstalled(kubeconfigPath string) (bool, error) {
	discoveryClient, err := kube.NewDiscoveryClient(kubeconfigPath)
	if err != nil {
		return false, fmt.Errorf("create discovery client: %w", err)
	}

	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		return false, fmt.Errorf("discover api groups: %w", err)
	}

	hasKustomize := false
	hasSource := false
	for _, group := range groups.Groups {
		switch group.Name {
		case "kustomize.toolkit.fluxcd.io":
			hasKustomize = true
		case "source.toolkit.fluxcd.io":
			hasSource = true
		}
	}

	return hasKustomize && hasSource, nil
}

func fluxGitRepositoryManifest(repoURL, branch string) []byte {
	return []byte(fmt.Sprintf(`apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 1m
  url: %q
  ref:
    branch: %q
`, repoURL, branch))
}

func fluxKustomizationsManifest(pathPrefix string) []byte {
	type fluxKustomization struct {
		Name      string
		Path      string
		DependsOn []string
		Wait      bool
	}

	items := []fluxKustomization{
		{Name: "helm-repositories", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/helm-repositories"), Wait: true},
		{Name: "namespaces", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/namespaces"), Wait: true},
		{Name: "crds", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/crds"), Wait: true},
		{Name: "helm-releases", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/helm-releases"), Wait: true, DependsOn: []string{"helm-repositories", "namespaces", "crds", "headlamp"}},
		{Name: "crossplane", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/crossplane"), DependsOn: []string{"helm-releases"}},
		{Name: "gateway", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/gateway"), DependsOn: []string{"namespaces"}},
		{Name: "policy-reporter", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/policy-reporter"), DependsOn: []string{"helm-releases"}},
		{Name: "trivy-dashboard", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/trivy-dashboard"), DependsOn: []string{"helm-releases"}},
		{Name: "headlamp", Path: fluxRepoPath(pathPrefix, "2-addons/manifests/headlamp"), DependsOn: []string{"namespaces"}},
	}

	var builder strings.Builder
	for index, item := range items {
		if index > 0 {
			builder.WriteString("---\n")
		}
		builder.WriteString("apiVersion: kustomize.toolkit.fluxcd.io/v1\n")
		builder.WriteString("kind: Kustomization\n")
		builder.WriteString("metadata:\n")
		builder.WriteString(fmt.Sprintf("  name: %s\n", item.Name))
		builder.WriteString("  namespace: flux-system\n")
		builder.WriteString("spec:\n")
		builder.WriteString("  interval: 10m\n")
		builder.WriteString(fmt.Sprintf("  path: %s\n", item.Path))
		builder.WriteString("  prune: true\n")
		if item.Wait {
			builder.WriteString("  wait: true\n")
		}
		builder.WriteString("  sourceRef:\n")
		builder.WriteString("    kind: GitRepository\n")
		builder.WriteString("    name: flux-system\n")
		if item.Name == "helm-releases" || item.Name == "gateway" || item.Name == "headlamp" {
			builder.WriteString("  postBuild:\n")
			builder.WriteString("    substituteFrom:\n")
			builder.WriteString("      - kind: ConfigMap\n")
			builder.WriteString(fmt.Sprintf("        name: %s\n", fluxPlatformConfigName))
		}
		if len(item.DependsOn) > 0 {
			builder.WriteString("  dependsOn:\n")
			for _, dependency := range item.DependsOn {
				builder.WriteString(fmt.Sprintf("    - name: %s\n", dependency))
			}
		}
	}
	return []byte(builder.String())
}

func fluxRepoPath(prefix, relativePath string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" || trimmed == "." {
		return "./" + relativePath
	}
	trimmed = strings.TrimSuffix(trimmed, "/")
	if strings.HasPrefix(trimmed, "./") {
		return trimmed + "/" + relativePath
	}
	return trimmed + "/" + relativePath
}

func fluxPlatformConfigManifest(publicConfig PublicDomainConfig) []byte {
	return []byte(fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: flux-system
data:
  SHOULDERS_DEX_HOST: %q
  SHOULDERS_GRAFANA_HOST: %q
  SHOULDERS_HEADLAMP_HOST: %q
  SHOULDERS_REPORTER_HOST: %q
  SHOULDERS_PROMETHEUS_HOST: %q
  SHOULDERS_ALERTMANAGER_HOST: %q
  SHOULDERS_HUBBLE_HOST: %q
  SHOULDERS_DEX_TLS_CERT_B64: %q
  SHOULDERS_DEX_TLS_KEY_B64: %q
  SHOULDERS_DEX_CA_CERT_B64: %q
`,
		fluxPlatformConfigName,
		publicConfig.DexHost,
		publicConfig.GrafanaHost,
		publicConfig.HeadlampHost,
		publicConfig.ReporterHost,
		publicConfig.PrometheusHost,
		publicConfig.AlertmanagerHost,
		publicConfig.HubbleHost,
		base64.StdEncoding.EncodeToString([]byte(publicConfig.TLS.CertPEM)),
		base64.StdEncoding.EncodeToString([]byte(publicConfig.TLS.KeyPEM)),
		base64.StdEncoding.EncodeToString([]byte(publicConfig.TLS.CAPEM)),
	))
}

func waitForFluxKustomizationsDeleted(ctx context.Context, kubeconfigPath string) error {
	client, err := kube.NewDynamicClient(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("create dynamic client: %w", err)
	}

	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		for _, name := range []string{"helm-repositories", "namespaces", "crds", "helm-releases", "crossplane", "gateway", "policy-reporter", "trivy-dashboard", "headlamp"} {
			if _, err := client.Resource(fluxKustomizationGVR).Namespace("flux-system").Get(ctx, name, metav1.GetOptions{}); err == nil {
				return false, nil
			}
		}
		return true, nil
	})
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
