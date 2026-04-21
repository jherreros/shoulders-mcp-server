package kube

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var contextOverride string

func SetContextOverride(contextName string) {
	contextOverride = contextName
}

func configLoadingRules(kubeconfig string) *clientcmd.ClientConfigLoadingRules {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
		return rules
	}
	if kubeConfigPaths := os.Getenv("KUBE_CONFIG_PATHS"); kubeConfigPaths != "" {
		rules.Precedence = filepath.SplitList(kubeConfigPaths)
	}
	return rules
}

func NewRestConfig(kubeconfig string) (*rest.Config, error) {
	rules := configLoadingRules(kubeconfig)
	overrides := &clientcmd.ConfigOverrides{}
	if contextOverride != "" {
		overrides.CurrentContext = contextOverride
	}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	// Raise the client rate limits from the defaults (5 QPS / 10 burst)
	// which are too low for the burst of API calls during bootstrap.
	config.QPS = 50
	config.Burst = 100
	return config, nil
}

func SwitchContext(kubeconfigPath, contextName string) error {
	rules := configLoadingRules(kubeconfigPath)
	config, err := rules.Load()
	if err != nil {
		return err
	}

	if _, ok := config.Contexts[contextName]; !ok {
		return fmt.Errorf("context %q does not exist", contextName)
	}

	config.CurrentContext = contextName

	return clientcmd.ModifyConfig(rules, *config, true)
}

func CurrentContext(kubeconfigPath string) (string, error) {
	rules := configLoadingRules(kubeconfigPath)
	config, err := rules.Load()
	if err != nil {
		return "", err
	}
	if contextOverride != "" {
		return contextOverride, nil
	}
	return config.CurrentContext, nil
}

func ListContexts(kubeconfigPath string) ([]string, error) {
	rules := configLoadingRules(kubeconfigPath)
	config, err := rules.Load()
	if err != nil {
		return nil, err
	}
	contexts := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}
	return contexts, nil
}

// IsShouldersContext returns true if the current kubeconfig context belongs
// to a vind cluster created by Shoulders (context name starts with "vcluster-docker_").
func IsShouldersContext(kubeconfigPath string) (bool, string, error) {
	ctx, err := CurrentContext(kubeconfigPath)
	if err != nil {
		return false, "", err
	}
	if strings.HasPrefix(ctx, "vcluster-docker_") {
		return true, ctx, nil
	}
	return false, ctx, nil
}
