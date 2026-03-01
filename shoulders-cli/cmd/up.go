package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/jherreros/shoulders/shoulders-cli/internal/flux"
	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"github.com/jherreros/shoulders/shoulders-cli/internal/manifests"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var upClusterName string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create the local cluster and install platform addons",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := bootstrap.EnsureKindCluster(upClusterName, manifests.KindConfig); err != nil {
			return fmt.Errorf("failed to create kind cluster: %w", err)
		}
		if err := bootstrap.EnsureCilium(kubeconfig); err != nil {
			return fmt.Errorf("failed to install cilium: %w", err)
		}
		if err := bootstrap.EnsureFlux(context.Background(), kubeconfig,
			manifests.FluxGitRepository,
			manifests.FluxKustomizations,
		); err != nil {
			return fmt.Errorf("failed to install flux: %w", err)
		}

		spinner, err := pterm.DefaultSpinner.Start("Waiting for Flux to reconcile")
		if err != nil {
			return err
		}
		if err := waitForFlux(spinner); err != nil {
			spinner.Fail("Flux reconciliation failed")
			return err
		}
		spinner.Success("Flux reconciliation complete")

		if err := bootstrap.RestartCiliumWorkloads(kubeconfig); err != nil {
			return fmt.Errorf("failed to restart cilium after flux reconciliation: %w", err)
		}
		if err := bootstrap.WaitForDeploymentReady(kubeconfig, "dex", "dex", 10*time.Minute); err != nil {
			return fmt.Errorf("failed waiting for dex deployment: %w", err)
		}
		if err := bootstrap.ConfigureAPIServerOIDC(upClusterName, kubeconfig, manifests.AuthenticationConfig); err != nil {
			return fmt.Errorf("failed to configure kube-apiserver OIDC: %w", err)
		}
		return nil
	},
}

func waitForFlux(spinner *pterm.SpinnerPrinter) error {
	ctx := context.Background()
	client, err := kube.NewDynamicClient(kubeconfig)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ticker.C:
			ready, pending, err := flux.AllKustomizationsReady(ctx, client, "flux-system")
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
			spinner.UpdateText(fmt.Sprintf("Waiting for Flux: %v", pending))
		case <-timeout:
			return fmt.Errorf("timed out waiting for Flux Kustomizations")
		}
	}
}

func init() {
	upCmd.Flags().StringVar(&upClusterName, "name", bootstrap.DefaultClusterName, "Name of the kind cluster")
}
