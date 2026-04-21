package cmd

import (
	"context"
	"fmt"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/jherreros/shoulders/shoulders-cli/internal/config"
	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"github.com/jherreros/shoulders/shoulders-cli/internal/manifests"
	"github.com/spf13/cobra"
)

var startClusterName string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a previously stopped vind cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		if currentConfig.Provider() == config.ProviderExisting {
			if err := bootstrap.EnsureExistingCluster(cmd.Context(), kubeconfig); err != nil {
				return err
			}
			fmt.Println("Existing cluster context is reachable")
			return nil
		}

		clusterName := configuredClusterName(cmd, "name", startClusterName)
		if err := bootstrap.StartVindCluster(cmd.Context(), clusterName); err != nil {
			return err
		}
		fmt.Printf("Cluster %q started, waiting for API server...\n", clusterName)

		if err := kube.WaitForAPIServer(context.Background(), kubeconfig); err != nil {
			return fmt.Errorf("waiting for API server: %w", err)
		}

		// Re-apply Gateway API CRDs so that any version-served flags
		// required by Cilium are up to date (e.g. TLSRoute v1alpha2).
		if err := kube.ApplyManifest(cmd.Context(), kubeconfig, manifests.GatewayAPICRDs, ""); err != nil {
			return fmt.Errorf("re-applying gateway api crds: %w", err)
		}

		if currentConfig.CiliumEnabled() {
			if err := bootstrap.RestartCiliumWorkloads(kubeconfig); err != nil {
				return fmt.Errorf("restarting cilium: %w", err)
			}
		}

		fmt.Printf("Cluster %q ready\n", clusterName)
		return nil
	},
}

func init() {
	startCmd.Flags().StringVar(&startClusterName, "name", bootstrap.DefaultClusterName, "Name of the vind cluster")
}
