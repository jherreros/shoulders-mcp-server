package cmd

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"github.com/spf13/cobra"
)

const (
	headlampNamespace = "headlamp"
	headlampService   = "headlamp"
)

var headlampCmd = &cobra.Command{
	Use:   "headlamp",
	Short: "Open Headlamp UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		const headlampOIDCURL = "http://headlamp.localhost"

		if isHostPortReachable("headlamp.localhost", "80", 1500*time.Millisecond) {
			cmd.Printf("Opening Headlamp via Gateway OIDC at %s\n", headlampOIDCURL)
			cmd.Printf("Sign in with Dex users (for example: admin@example.com / password).\n")
			if err := openBrowser(headlampOIDCURL); err == nil {
				return nil
			}
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		cmd.Printf("Headlamp Gateway URL not reachable; falling back to local port-forward on http://localhost:4466\n")

		stopCh, _, err := kube.PortForwardService(ctx, kubeconfig, headlampNamespace, headlampService, 4466, 80)
		if err != nil {
			return err
		}
		defer close(stopCh)

		go func() {
			time.Sleep(2 * time.Second)
			_ = openBrowser("http://localhost:4466")
		}()

		<-ctx.Done()
		return nil
	},
}
