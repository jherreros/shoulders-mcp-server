package cmd

import (
	"fmt"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/spf13/cobra"
)

var startClusterName string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a previously stopped vind cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := bootstrap.StartVindCluster(cmd.Context(), startClusterName); err != nil {
			return err
		}
		fmt.Printf("Cluster %q started\n", startClusterName)
		return nil
	},
}

func init() {
	startCmd.Flags().StringVar(&startClusterName, "name", bootstrap.DefaultClusterName, "Name of the vind cluster")
}
