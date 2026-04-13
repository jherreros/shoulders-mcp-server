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
	"Create vind cluster",
	"Install Cilium CNI",
	"Install Flux CD",
	"Reconcile Flux kustomizations",
	"Wait for platform deployments",
	"Configure gateway routes",
	"Validate cluster status",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create the local cluster and install platform addons",
	RunE: func(cmd *cobra.Command, args []string) error {
		tracker := tui.NewPhaseTracker(upPhases, upVerbose)
		defer tracker.Stop()

		// Phase 1: vind cluster
		tracker.Start(verboseDetail("creating vind cluster %q using embedded config", upClusterName))
		if err := bootstrap.EnsureVindCluster(cmd.Context(), upClusterName, manifests.VindConfig, manifests.AuthenticationConfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to create vind cluster: %w", err)
		}
		tracker.Complete()

		// Phase 2: Cilium
		tracker.Start(verboseDetail("installing Gateway API CRDs and Cilium Helm chart with gatewayAPI"))
		// Install Gateway API CRDs before Cilium so the operator can
		// register the GatewayClass controller on startup.
		if err := kube.ApplyManifest(cmd.Context(), kubeconfig, manifests.GatewayAPICRDs, ""); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to install gateway api crds: %w", err)
		}
		if err := bootstrap.EnsureCilium(kubeconfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to install cilium: %w", err)
		}
		if err := bootstrap.RestartStuckPods(kubeconfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to restart stuck pods: %w", err)
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
		// Suspend the Flux-managed Cilium HelmRelease so the helm-controller
		// does not override the CLI-managed Cilium installation with values
		// from the Git repo that may be incompatible with vind.
		if err := bootstrap.SuspendCiliumHelmRelease(kubeconfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to suspend cilium helmrelease: %w", err)
		}
		tracker.Complete()

		// Phase 4: Flux reconciliation
		tracker.Start("waiting for kustomizations...")
		if err := waitForFluxTUI(tracker); err != nil {
			tracker.Fail(err.Error())
			return err
		}
		// Delete any pods stuck in ContainerCreating from the initial
		// Flux reconciliation. The burst of pod creation can overwhelm
		// Cilium's endpoint API, leaving pods in exponential backoff.
		if err := bootstrap.RestartStuckPods(kubeconfig); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to restart stuck pods: %w", err)
		}
		tracker.Complete()

		// Phase 5: Platform deployments
		deployments := []struct{ ns, name string }{
			{"dex", "dex"},
			{"headlamp", "headlamp"},
			{"observability", "kube-prometheus-stack-grafana"},
			{"policy-reporter", "policy-reporter-ui"},
		}
		tracker.Start(verboseDetail("waiting for %d deployments: dex, headlamp, grafana, policy-reporter-ui", len(deployments)))
		for _, d := range deployments {
			tracker.UpdateDetail(fmt.Sprintf("waiting for %s/%s", d.ns, d.name))
			if err := bootstrap.WaitForDeploymentReady(kubeconfig, d.ns, d.name, 10*time.Minute); err != nil {
				tracker.Fail(fmt.Sprintf("%s/%s not ready", d.ns, d.name))
				return fmt.Errorf("failed waiting for %s deployment: %w", d.name, err)
			}
		}
		tracker.Complete()

		// Phase 6: Gateway routes
		tracker.Start(verboseDetail("resolving HTTPRoutes"))
		routes := []struct{ ns, name string }{
			{"dex", "dex"},
			{"headlamp", "headlamp"},
			{"observability", "grafana"},
			{"policy-reporter", "policy-reporter"},
		}
		for _, r := range routes {
			tracker.UpdateDetail(fmt.Sprintf("waiting for %s/%s HTTPRoute", r.ns, r.name))
			if err := bootstrap.WaitForHTTPRouteResolved(kubeconfig, r.ns, r.name, 5*time.Minute); err != nil {
				tracker.Fail(fmt.Sprintf("%s route not resolved", r.name))
				return fmt.Errorf("failed waiting for %s route: %w", r.name, err)
			}
		}
		tracker.Complete()

		// Phase 7: Status validation
		tracker.Start(verboseDetail("waiting for shoulders status to report all systems healthy"))
		if err := waitForHealthyStatus(cmd.Context(), 5*time.Minute); err != nil {
			tracker.Fail(err.Error())
			return fmt.Errorf("failed to validate cluster status: %w", err)
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
	// A fully cold Docker engine must pull the complete addon image set,
	// which can push the initial Flux reconciliation past 10 minutes.
	timeout := time.After(20 * time.Minute)

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
			// Later phases already wait on the concrete resources backed by
			// these kustomizations, so we don't need to block here once only
			// the crossplane/gateway follow-up reconciliations remain.
			allDeferred := true
			for _, p := range pending {
				if p != "crossplane" && p != "gateway" {
					allDeferred = false
					break
				}
			}
			if allDeferred && len(pending) > 0 {
				return nil
			}
			tracker.UpdateDetail(fmt.Sprintf("pending: %s", strings.Join(pending, ", ")))
		case <-timeout:
			return fmt.Errorf("timed out waiting for Flux Kustomizations")
		}
	}
}

func waitForHealthyStatus(ctx context.Context, timeout time.Duration) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastErr error
	for {
		summary, err := gatherStatus(deadlineCtx)
		if err != nil {
			lastErr = err
		} else {
			podsHealthy := summary.TotalPods > 0 && summary.HealthyPods == summary.TotalPods
			if summary.NodesReady && podsHealthy && summary.FluxReady && summary.XPlaneReady && summary.GatewayReady {
				return nil
			}
		}

		select {
		case <-deadlineCtx.Done():
			if lastErr != nil {
				return lastErr
			}
			return fmt.Errorf("timed out waiting for shoulders status to report all systems healthy")
		case <-ticker.C:
		}
	}
}

func init() {
	upCmd.Flags().StringVar(&upClusterName, "name", bootstrap.DefaultClusterName, "Name of the vind cluster")
	upCmd.Flags().BoolVarP(&upVerbose, "verbose", "v", false, "Show detailed progress information for each phase")
}
