package cmd

import (
	"fmt"
	"os"

	"github.com/jherreros/shoulders/shoulders-cli/internal/config"
	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"github.com/jherreros/shoulders/shoulders-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Commands that do not require a shoulders cluster context.
var skipClusterCheck = map[string]bool{
	"up":      true,
	"update":  true,
	"cluster": true,
	"start":   true,
	"stop":    true,
	"help":    true,
	"version": true,
}

var (
	rootCmd = &cobra.Command{
		Use:     "shoulders",
		Short:   "Developer CLI for the Shoulders IDP",
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			currentConfig = cfg

			if err := ensureShouldersCluster(cmd); err != nil {
				return err
			}
			return nil
		},
	}
	currentConfig *config.Config
	kubeconfig    string
	outputFormat  string
)

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", string(output.Table), "Output format: table|json|yaml")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(workspaceCmd)
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(infraCmd)
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(portalCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(reporterCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
}

func initConfig() {
	viper.SetEnvPrefix("SHOULDERS")
	viper.AutomaticEnv()
}

func outputOption() (output.Format, error) {
	return output.ParseFormat(outputFormat)
}

// ensureShouldersCluster verifies the current kubeconfig context belongs to
// a Shoulders-managed vind cluster. Commands in skipClusterCheck are exempt.
func ensureShouldersCluster(cmd *cobra.Command) error {
	// Walk up to find the top-level subcommand name.
	name := rootCmdName(cmd)
	if skipClusterCheck[name] {
		return nil
	}

	ok, ctx, err := kube.IsShouldersContext(kubeconfig)
	if err != nil {
		return fmt.Errorf("cannot determine cluster context: %w", err)
	}
	if !ok {
		return fmt.Errorf("current context %q is not a Shoulders cluster; switch with 'shoulders cluster use <name>' or run 'shoulders up'", ctx)
	}
	return nil
}

func rootCmdName(cmd *cobra.Command) string {
	for cmd.HasParent() && cmd.Parent().HasParent() {
		cmd = cmd.Parent()
	}
	return cmd.Name()
}
