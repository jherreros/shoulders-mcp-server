package cmd

import (
	"fmt"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/jherreros/shoulders/shoulders-cli/internal/config"
	"github.com/spf13/cobra"
)

var downClusterName string

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Delete the local vind cluster or uninstall Shoulders from an existing cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		if currentConfig.Provider() == config.ProviderExisting {
			if err := bootstrap.UninstallShouldersFlux(cmd.Context(), kubeconfig, currentConfig.FluxPathPrefix()); err != nil {
				return err
			}
			if err := bootstrap.CleanupKyvernoWebhooks(kubeconfig); err != nil {
				return fmt.Errorf("cleanup kyverno webhooks: %w", err)
			}
			if currentConfig.CiliumEnabled() {
				if err := bootstrap.UninstallCilium(kubeconfig); err != nil {
					return fmt.Errorf("uninstall cilium: %w", err)
				}
			}
			if err := bootstrap.WaitForNamespacesDeleted(cmd.Context(), kubeconfig); err != nil {
				return fmt.Errorf("wait for platform namespaces to terminate: %w", err)
			}
			fmt.Println("Shoulders platform resources removed from the current cluster")
			return nil
		}
		clusterName := configuredClusterName(cmd, "name", downClusterName)
		return bootstrap.DeleteVindCluster(cmd.Context(), clusterName)
	},
}

func init() {
	downCmd.Flags().StringVar(&downClusterName, "name", bootstrap.DefaultClusterName, "Name of the vind cluster to delete")
}
