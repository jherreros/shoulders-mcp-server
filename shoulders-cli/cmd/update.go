package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

const (
	githubOwner = "jherreros"
	githubRepo  = "shoulders"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for a new CLI version and update if available",
	RunE: func(cmd *cobra.Command, args []string) error {
		current := Version
		if current == "dev" {
			return fmt.Errorf("cannot update a dev build; install a released version first")
		}

		spinner, _ := pterm.DefaultSpinner.Start("Checking for updates...")
		release, err := fetchLatestRelease(cmd.Context())
		if err != nil {
			spinner.Fail("Failed to check for updates")
			return fmt.Errorf("failed to fetch latest release: %w", err)
		}

		latest := strings.TrimPrefix(release.TagName, "v")
		current = strings.TrimPrefix(current, "v")

		if latest == current {
			spinner.Success(fmt.Sprintf("Already up to date (%s)", release.TagName))
			return nil
		}

		spinner.UpdateText(fmt.Sprintf("New version available: %s → %s", current, latest))

		assetName := fmt.Sprintf("shoulders-%s-%s", runtime.GOOS, runtime.GOARCH)
		var downloadURL string
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
		if downloadURL == "" {
			spinner.Fail("No binary found for your platform")
			return fmt.Errorf("no release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
		}

		spinner.UpdateText(fmt.Sprintf("Downloading %s...", release.TagName))

		binary, err := downloadAsset(cmd.Context(), downloadURL)
		if err != nil {
			spinner.Fail("Download failed")
			return fmt.Errorf("failed to download update: %w", err)
		}

		execPath, err := os.Executable()
		if err != nil {
			spinner.Fail("Cannot determine executable path")
			return err
		}

		if err := replaceExecutable(execPath, binary); err != nil {
			spinner.Fail("Update failed")
			return fmt.Errorf("failed to replace binary: %w", err)
		}

		spinner.Success(fmt.Sprintf("Updated to %s", release.TagName))
		return nil
	},
}

func fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func downloadAsset(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func replaceExecutable(path string, newBinary []byte) error {
	tmpPath := path + ".new"
	if err := os.WriteFile(tmpPath, newBinary, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
