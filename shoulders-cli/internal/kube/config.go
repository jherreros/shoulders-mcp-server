package kube

import (
	"fmt"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewRestConfig(kubeconfig string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	return clientConfig.ClientConfig()
}

func SwitchContext(kubeconfigPath, contextName string) error {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}
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

// IsShouldersContext returns true if the current kubeconfig context belongs
// to a vind cluster created by Shoulders (context name starts with "vcluster-docker_shoulders").
func IsShouldersContext(kubeconfigPath string) (bool, string, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}
	config, err := rules.Load()
	if err != nil {
		return false, "", err
	}
	ctx := config.CurrentContext
	if strings.HasPrefix(ctx, "vcluster-docker_shoulders") {
		return true, ctx, nil
	}
	return false, ctx, nil
}
