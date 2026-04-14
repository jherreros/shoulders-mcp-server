package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var skillReferenceFiles = []string{
	"references/cli-reference.md",
	"references/examples.md",
	"references/mcp-server.md",
}

var skillWorkspace bool

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage the Shoulders agent skill",
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Shoulders skill for AI agents (VS Code Copilot)",
	Long: `Download and install the Shoulders SKILL.md so AI agents can deploy
and manage applications on the platform.

By default installs globally to ~/.copilot/skills/shoulders/ (available
across all workspaces). Use --workspace to install into the current
project's .github/skills/shoulders/ directory instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var destDir string
		if skillWorkspace {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine working directory: %w", err)
			}
			destDir = filepath.Join(cwd, ".github", "skills", "shoulders")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}
			destDir = filepath.Join(home, ".copilot", "skills", "shoulders")
		}

		spinner, _ := pterm.DefaultSpinner.Start("Installing Shoulders skill...")

		// Create destination directory tree (including references/).
		refsDir := filepath.Join(destDir, "references")
		if err := os.MkdirAll(refsDir, 0o755); err != nil {
			spinner.Fail("Failed to create directory")
			return fmt.Errorf("cannot create directory %s: %w", refsDir, err)
		}

		// Download SKILL.md.
		baseURL := "https://raw.githubusercontent.com/" + githubOwner + "/" + githubRepo + "/main/.github/skills/shoulders/"
		if err := downloadFile(cmd.Context(), baseURL+"SKILL.md", filepath.Join(destDir, "SKILL.md")); err != nil {
			spinner.Fail("Failed to download SKILL.md")
			return err
		}

		// Download reference files.
		for _, ref := range skillReferenceFiles {
			if err := downloadFile(cmd.Context(), baseURL+ref, filepath.Join(destDir, ref)); err != nil {
				spinner.Fail(fmt.Sprintf("Failed to download %s", ref))
				return err
			}
		}

		spinner.Success(fmt.Sprintf("Skill installed to %s", destDir))

		if skillWorkspace {
			cmd.Println("The skill is available in this workspace. Invoke it with /shoulders in VS Code Copilot chat.")
		} else {
			cmd.Println("The skill is available across all workspaces. Invoke it with /shoulders in VS Code Copilot chat.")
		}
		return nil
	},
}

func init() {
	skillInstallCmd.Flags().BoolVar(&skillWorkspace, "workspace", false, "Install into the current project (.github/skills/shoulders/) instead of globally")
	skillCmd.AddCommand(skillInstallCmd)
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dest, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write %s: %w", dest, err)
	}
	return nil
}
