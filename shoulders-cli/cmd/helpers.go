package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var deferredFluxKustomizations = map[string]bool{
	"helm-releases":   true,
	"crossplane":      true,
	"gateway":         true,
	"policy-reporter": true,
	"trivy-dashboard": true,
}

var namespaceOverride string

func registerNamespaceFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&namespaceOverride, "namespace", "n", "", "Workspace namespace to target")
}

func currentNamespace() (string, error) {
	if namespaceOverride != "" {
		return namespaceOverride, nil
	}
	if currentConfig == nil || currentConfig.CurrentWorkspace == "" {
		return "", errors.New("no active workspace: run 'shoulders workspace use <name>' or pass --namespace")
	}
	return currentConfig.CurrentWorkspace, nil
}

func gatewayChecksRequired() bool {
	return currentConfig == nil || currentConfig.CiliumEnabled()
}

func onlyDeferredFluxKustomizations(pending []string) bool {
	if len(pending) == 0 {
		return false
	}
	for _, name := range pending {
		if !deferredFluxKustomizations[name] {
			return false
		}
	}
	return true
}
