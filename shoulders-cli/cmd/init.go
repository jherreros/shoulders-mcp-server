package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jherreros/shoulders/shoulders-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	initProvider string
	initForce    bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write an example Shoulders config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := config.ExampleYAML(initProvider)
		if err != nil {
			return err
		}

		targetPath := loadedConfigPath
		if targetPath == "" {
			targetPath, err = config.Path(configFile)
			if err != nil {
				return err
			}
		}

		if !initForce {
			if _, err := os.Stat(targetPath); err == nil {
				return fmt.Errorf("config file %q already exists; use --force to overwrite", targetPath)
			} else if !os.IsNotExist(err) {
				return err
			}
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return err
		}

		fmt.Printf("Wrote example config to %s\n", targetPath)
		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initProvider, "provider", config.ProviderVind, "Cluster provider to scaffold: vind|existing")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite the target config file if it already exists")
}
