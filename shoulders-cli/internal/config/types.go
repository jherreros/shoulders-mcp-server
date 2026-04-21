package config

import (
	"fmt"
	"strings"
)

const (
	ProviderVind     = "vind"
	ProviderExisting = "existing"

	DefaultClusterName      = "shoulders"
	DefaultCiliumVersion    = "1.19.2"
	DefaultFluxRepoURL      = "https://github.com/jherreros/shoulders.git"
	DefaultFluxBranch       = "main"
	DefaultDexHost          = "dex.127.0.0.1.sslip.io"
	DefaultGrafanaHost      = "grafana.localhost"
	DefaultHeadlampHost     = "headlamp.localhost"
	DefaultReporterHost     = "reporter.localhost"
	DefaultPrometheusHost   = "prometheus.localhost"
	DefaultAlertmanagerHost = "alertmanager.localhost"
	DefaultHubbleHost       = "hubble.localhost"
)

type Config struct {
	CurrentWorkspace string         `yaml:"current_workspace,omitempty" json:"current_workspace,omitempty"`
	Cluster          ClusterConfig  `yaml:"cluster,omitempty" json:"cluster,omitempty"`
	Platform         PlatformConfig `yaml:"platform,omitempty" json:"platform,omitempty"`
}

type ClusterConfig struct {
	Provider   string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
	Kubeconfig string `yaml:"kubeconfig,omitempty" json:"kubeconfig,omitempty"`
	Context    string `yaml:"context,omitempty" json:"context,omitempty"`
}

type PlatformConfig struct {
	Domain string       `yaml:"domain,omitempty" json:"domain,omitempty"`
	Cilium CiliumConfig `yaml:"cilium,omitempty" json:"cilium,omitempty"`
	Flux   FluxConfig   `yaml:"flux,omitempty" json:"flux,omitempty"`
}

type CiliumConfig struct {
	Enabled *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type FluxConfig struct {
	GitRepository GitRepositoryConfig `yaml:"gitRepository,omitempty" json:"gitRepository,omitempty"`
	PathPrefix    string              `yaml:"pathPrefix,omitempty" json:"pathPrefix,omitempty"`
}

type GitRepositoryConfig struct {
	URL    string `yaml:"url,omitempty" json:"url,omitempty"`
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.ApplyDefaults()
	return cfg
}

func (cfg *Config) ApplyDefaults() {
	if cfg.Cluster.Provider == "" {
		cfg.Cluster.Provider = ProviderVind
	}
	if cfg.Cluster.Name == "" {
		cfg.Cluster.Name = DefaultClusterName
	}
	if cfg.Platform.Cilium.Version == "" {
		cfg.Platform.Cilium.Version = DefaultCiliumVersion
	}
	if cfg.Platform.Cilium.Enabled == nil {
		enabled := cfg.Cluster.Provider == ProviderVind
		cfg.Platform.Cilium.Enabled = &enabled
	}
	if cfg.Platform.Flux.GitRepository.URL == "" {
		cfg.Platform.Flux.GitRepository.URL = DefaultFluxRepoURL
	}
	if cfg.Platform.Flux.GitRepository.Branch == "" {
		cfg.Platform.Flux.GitRepository.Branch = DefaultFluxBranch
	}
	if cfg.Platform.Flux.PathPrefix == "" {
		cfg.Platform.Flux.PathPrefix = "."
	}
}

func (cfg *Config) Validate() error {
	if domain := cfg.Domain(); domain != "" {
		if strings.Contains(domain, "://") || strings.Contains(domain, "/") {
			return fmt.Errorf("platform.domain must be a host suffix without a scheme or path")
		}
	}
	switch cfg.Provider() {
	case ProviderVind, ProviderExisting:
		return nil
	default:
		return fmt.Errorf("unsupported cluster provider %q", cfg.Cluster.Provider)
	}
}

func (cfg *Config) Provider() string {
	if cfg == nil {
		return ProviderVind
	}
	if cfg.Cluster.Provider == "" {
		return ProviderVind
	}
	return cfg.Cluster.Provider
}

func (cfg *Config) ClusterName() string {
	if cfg == nil || cfg.Cluster.Name == "" {
		return DefaultClusterName
	}
	return cfg.Cluster.Name
}

func (cfg *Config) CiliumEnabled() bool {
	if cfg == nil || cfg.Platform.Cilium.Enabled == nil {
		return cfg.Provider() == ProviderVind
	}
	return *cfg.Platform.Cilium.Enabled
}

func (cfg *Config) CiliumVersion() string {
	if cfg == nil || cfg.Platform.Cilium.Version == "" {
		return DefaultCiliumVersion
	}
	return cfg.Platform.Cilium.Version
}

func (cfg *Config) FluxRepositoryURL() string {
	if cfg == nil || cfg.Platform.Flux.GitRepository.URL == "" {
		return DefaultFluxRepoURL
	}
	return cfg.Platform.Flux.GitRepository.URL
}

func (cfg *Config) FluxRepositoryBranch() string {
	if cfg == nil || cfg.Platform.Flux.GitRepository.Branch == "" {
		return DefaultFluxBranch
	}
	return cfg.Platform.Flux.GitRepository.Branch
}

func (cfg *Config) FluxPathPrefix() string {
	if cfg == nil || cfg.Platform.Flux.PathPrefix == "" {
		return "."
	}
	return cfg.Platform.Flux.PathPrefix
}

func (cfg *Config) Domain() string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Platform.Domain)
}

func (cfg *Config) HasCustomDomain() bool {
	return cfg.Domain() != ""
}

func (cfg *Config) DexHost() string {
	if domain := cfg.Domain(); domain != "" {
		return "dex." + domain
	}
	return DefaultDexHost
}

func (cfg *Config) DexURL() string {
	return "https://" + cfg.DexHost()
}

func (cfg *Config) GrafanaHost() string {
	if domain := cfg.Domain(); domain != "" {
		return "grafana." + domain
	}
	return DefaultGrafanaHost
}

func (cfg *Config) GrafanaURL() string {
	return "http://" + cfg.GrafanaHost()
}

func (cfg *Config) HeadlampHost() string {
	if domain := cfg.Domain(); domain != "" {
		return "headlamp." + domain
	}
	return DefaultHeadlampHost
}

func (cfg *Config) HeadlampURL() string {
	return "http://" + cfg.HeadlampHost()
}

func (cfg *Config) ReporterHost() string {
	if domain := cfg.Domain(); domain != "" {
		return "reporter." + domain
	}
	return DefaultReporterHost
}

func (cfg *Config) ReporterURL() string {
	return "http://" + cfg.ReporterHost()
}

func (cfg *Config) PrometheusHost() string {
	if domain := cfg.Domain(); domain != "" {
		return "prometheus." + domain
	}
	return DefaultPrometheusHost
}

func (cfg *Config) AlertmanagerHost() string {
	if domain := cfg.Domain(); domain != "" {
		return "alertmanager." + domain
	}
	return DefaultAlertmanagerHost
}

func (cfg *Config) HubbleHost() string {
	if domain := cfg.Domain(); domain != "" {
		return "hubble." + domain
	}
	return DefaultHubbleHost
}

func ExampleYAML(provider string) (string, error) {
	providerValue := provider
	if providerValue == "" {
		providerValue = ProviderVind
	}
	if providerValue != ProviderVind && providerValue != ProviderExisting {
		return "", fmt.Errorf("unsupported cluster provider %q", providerValue)
	}

	ciliumEnabled := "true"
	contextHint := "  # Optional kube context override"
	if providerValue == ProviderExisting {
		ciliumEnabled = "false"
		contextHint = "current-context  # Required if you do not want to use the current kubeconfig context"
	}

	return fmt.Sprintf(
		"# Shoulders CLI configuration\n"+
			"# Pass this file with --config, or place it at ~/.shoulders/config.yaml.\n\n"+
			"current_workspace: \"\"\n\n"+
			"cluster:\n"+
			"  provider: %s\n"+
			"  name: shoulders\n"+
			"  kubeconfig: \"\"\n"+
			"  context: %s\n\n"+
			"platform:\n"+
			"  domain: \"\"\n"+
			"  cilium:\n"+
			"    enabled: %s\n"+
			"    version: %q\n"+
			"  flux:\n"+
			"    gitRepository:\n"+
			"      url: %q\n"+
			"      branch: %q\n"+
			"    pathPrefix: %q\n",
		providerValue,
		contextHint,
		ciliumEnabled,
		DefaultCiliumVersion,
		DefaultFluxRepoURL,
		DefaultFluxBranch,
		".",
	), nil
}
