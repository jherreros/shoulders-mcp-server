package cmd

import (
	"fmt"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/spf13/cobra"
)

var stopClusterName string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the local vind cluster without deleting it",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := bootstrap.StopVindCluster(cmd.Context(), stopClusterName); err != nil {
			return err
		}
		fmt.Printf("Cluster %q stopped\n", stopClusterName)
		return nil
	},
}

func init() {
	stopCmd.Flags().StringVar(&stopClusterName, "name", bootstrap.DefaultClusterName, "Name of the vind cluster")
}
