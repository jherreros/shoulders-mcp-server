package cmd

import (
	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/spf13/cobra"
)

var downClusterName string

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Delete the local vind cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		return bootstrap.DeleteVindCluster(cmd.Context(), downClusterName)
	},
}

func init() {
	downCmd.Flags().StringVar(&downClusterName, "name", bootstrap.DefaultClusterName, "Name of the vind cluster")
}
