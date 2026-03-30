package cmd

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"github.com/spf13/cobra"
)

var reporterCmd = &cobra.Command{
	Use:   "reporter",
	Short: "Open the Policy Reporter dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		const reporterURL = "http://reporter.localhost"

		if isHostPortReachable("reporter.localhost", "80", 1500*time.Millisecond) {
			cmd.Printf("Opening Policy Reporter at %s\n", reporterURL)
			if err := openBrowser(reporterURL); err == nil {
				return nil
			}
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		cmd.Printf("Policy Reporter Gateway URL not reachable; falling back to local port-forward on http://localhost:8082\n")

		stopCh, _, err := kube.PortForwardService(ctx, kubeconfig, "policy-reporter", "policy-reporter-ui", 8082, 8080)
		if err != nil {
			return err
		}
		defer close(stopCh)

		go func() {
			time.Sleep(2 * time.Second)
			_ = openBrowser("http://localhost:8082")
		}()

		<-ctx.Done()
		return nil
	},
}
