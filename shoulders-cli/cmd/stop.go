package cmd

import (
	"fmt"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/jherreros/shoulders/shoulders-cli/internal/config"
	"github.com/spf13/cobra"
)

var stopClusterName string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the local vind cluster without deleting it",
	RunE: func(cmd *cobra.Command, args []string) error {
		if currentConfig.Provider() == config.ProviderExisting {
			return fmt.Errorf("stop is not supported for provider=%s", currentConfig.Provider())
		}
		clusterName := configuredClusterName(cmd, "name", stopClusterName)
		if err := bootstrap.StopVindCluster(cmd.Context(), clusterName); err != nil {
			return err
		}
		fmt.Printf("Cluster %q stopped\n", clusterName)
		return nil
	},
}

func init() {
	stopCmd.Flags().StringVar(&stopClusterName, "name", bootstrap.DefaultClusterName, "Name of the vind cluster")
}
