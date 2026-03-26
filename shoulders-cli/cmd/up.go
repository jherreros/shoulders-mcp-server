package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/bootstrap"
	"github.com/jherreros/shoulders/shoulders-cli/internal/flux"
	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"github.com/jherreros/shoulders/shoulders-cli/internal/manifests"
	"github.com/jherreros/shoulders/shoulders-cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	upClusterName string
	upVerbose     bool
)

var upPhases = []string{
	"Create kind cluster",
	"Install Cilium CNI",
	"Install Flux CD",
	"Reconcile Flux kustomizations",
	"Wait for platform deployments",
	"Configure gateway routes",
	"Configure API server OIDC",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create the local cluster and install platform addons",
	RunE: func(cmd *cobra.Command, args []string) error {
		tracker := tui.NewPhaseTracker(upPhases, upVerbose)
		defer tracker.Stop()

		// Phase 1: Kind cluster
		tracker.Start(verboseDetail("creating kind cluster %q using embedded config", upClusterName))
		if err := bootstrap.EnsureKindCluster(upClusterName, manifests.KindConfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to create kind cluster: %w", err)
		}
		tracker.Complete()

		// Phase 2: Cilium
		tracker.Start(verboseDetail("installing Cilium Helm chart with gatewayAPI and kubeProxyReplacement"))
		if err := bootstrap.EnsureCilium(kubeconfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to install cilium: %w", err)
		}
		tracker.Complete()

		// Phase 3: Flux install
		tracker.Start(verboseDetail("downloading Flux install manifest and applying GitRepository + Kustomizations"))
		if err := bootstrap.EnsureFlux(context.Background(), kubeconfig,
			manifests.FluxGitRepository,
			manifests.FluxKustomizations,
		); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to install flux: %w", err)
		}
		tracker.Complete()

		// Phase 4: Flux reconciliation
		tracker.Start("waiting for kustomizations...")
		if err := waitForFluxTUI(tracker); err != nil {
			tracker.Fail(err.Error())
			return err
		}
		tracker.Complete()

		// Phase 5: Platform deployments
		deployments := []struct{ ns, name string }{
			{"dex", "dex"},
			{"headlamp", "headlamp"},
			{"observability", "kube-prometheus-stack-grafana"},
		}
		tracker.Start(verboseDetail("waiting for %d deployments: dex, headlamp, grafana", len(deployments)))
		for _, d := range deployments {
			tracker.UpdateDetail(fmt.Sprintf("waiting for %s/%s", d.ns, d.name))
			if err := bootstrap.WaitForDeploymentReady(kubeconfig, d.ns, d.name, 10*time.Minute); err != nil {
				tracker.Fail(fmt.Sprintf("%s/%s not ready", d.ns, d.name))
				return fmt.Errorf("failed waiting for %s deployment: %w", d.name, err)
			}
		}
		tracker.Complete()

		// Phase 6: Gateway routes
		tracker.Start(verboseDetail("restarting Cilium workloads and resolving HTTPRoutes"))
		if err := bootstrap.RestartCiliumWorkloads(kubeconfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to restart cilium after flux reconciliation: %w", err)
		}
		routes := []struct{ ns, name string }{
			{"dex", "dex"},
			{"headlamp", "headlamp"},
			{"observability", "grafana"},
		}
		for _, r := range routes {
			tracker.UpdateDetail(fmt.Sprintf("waiting for %s/%s HTTPRoute", r.ns, r.name))
			if err := bootstrap.WaitForHTTPRouteResolved(kubeconfig, r.ns, r.name, 5*time.Minute); err != nil {
				tracker.Fail(fmt.Sprintf("%s route not resolved", r.name))
				return fmt.Errorf("failed waiting for %s route: %w", r.name, err)
			}
		}
		tracker.Complete()

		// Phase 7: OIDC
		tracker.Start(verboseDetail("writing authentication-config to control-plane node and restarting kube-apiserver"))
		if err := bootstrap.ConfigureAPIServerOIDC(upClusterName, kubeconfig, manifests.AuthenticationConfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to configure kube-apiserver OIDC: %w", err)
		}
		tracker.Complete()

		fmt.Println()
		fmt.Println(tracker.Summary())
		fmt.Println()
		return nil
	},
}

// verboseDetail returns detail only when --verbose is set.
func verboseDetail(format string, a ...any) string {
	if !upVerbose {
		return ""
	}
	return fmt.Sprintf(format, a...)
}

func waitForFluxTUI(tracker *tui.PhaseTracker) error {
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
			tracker.UpdateDetail(fmt.Sprintf("pending: %s", strings.Join(pending, ", ")))
		case <-timeout:
			return fmt.Errorf("timed out waiting for Flux Kustomizations")
		}
	}
}

func init() {
	upCmd.Flags().StringVar(&upClusterName, "name", bootstrap.DefaultClusterName, "Name of the kind cluster")
	upCmd.Flags().BoolVarP(&upVerbose, "verbose", "v", false, "Show detailed progress information for each phase")
}
