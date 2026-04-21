package cmd

import (
	"fmt"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/jherreros/shoulders/shoulders-cli/internal/config"
	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"github.com/jherreros/shoulders/shoulders-cli/internal/output"
	"github.com/spf13/cobra"
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage target clusters and kube contexts",
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing clusters or kube contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			clusters []string
			err      error
		)
		if currentConfig.Provider() == config.ProviderExisting {
			clusters, err = kube.ListContexts(kubeconfig)
		} else {
			clusters, err = bootstrap.ListClusters()
		}
		if err != nil {
			return err
		}

		format, err := outputOption()
		if err != nil {
			return err
		}

		if format == output.Table {
			rows := [][]string{}
			for _, c := range clusters {
				rows = append(rows, []string{c})
			}
			return output.PrintTable([]string{"Name"}, rows)
		}

		payload, err := output.Render(clusters, format)
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
		return nil
	},
}

var clusterUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch context to another cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		contextName := name
		if currentConfig.Provider() != config.ProviderExisting {
			contextName = bootstrap.ContextPrefix + name
		}

		if err := kube.SwitchContext(kubeconfig, contextName); err != nil {
			return err
		}
		if currentConfig.Provider() == config.ProviderExisting {
			currentConfig.Cluster.Context = contextName
		} else {
			currentConfig.Cluster.Name = name
		}
		if err := saveCurrentConfig(); err != nil {
			return err
		}
		fmt.Printf("Switched to cluster %s (context: %s)\n", name, contextName)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterUseCmd)
}
