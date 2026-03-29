package bootstrap

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	authConfigPath          = "/etc/kubernetes/authn/authentication-config.yaml"
	defaultDexServiceIP     = "10.96.0.24"
	dexNamespace            = "dex"
	dexServiceName          = "dex"
	dexInternalHosts        = "dex.dex.svc.cluster.local dex.dex.svc"
	dexPublicHost           = "dex.127.0.0.1.sslip.io"
	apiserverRestartTimeout = 2 * time.Minute
)

func WaitForDeploymentReady(kubeconfig, namespace, name string, timeout time.Duration) error {
	clientset, err := kube.NewClientset(kubeconfig)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err == nil && deploymentReady(deployment) {
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("deployment %s/%s did not become ready within %s", namespace, name, timeout)
}

func ConfigureAPIServerOIDC(name, kubeconfig string, authConfig []byte) error {
	dexServiceIP, err := dexServiceIP(kubeconfig)
	if err != nil {
		return err
	}

	containerName := controlPlanePrefix + name

	exists, err := containerExists(context.Background(), containerName)
	if err != nil {
		return fmt.Errorf("check control-plane container: %w", err)
	}
	if !exists {
		return fmt.Errorf("control-plane container %q not found for cluster %q", containerName, name)
	}

	previousStartTime, err := apiserverObservedStartTime(kubeconfig)
	if err != nil {
		return fmt.Errorf("get kube-apiserver start time: %w", err)
	}

	authConfigEncoded := base64.StdEncoding.EncodeToString(authConfig)

	// In vind, the kube-apiserver is managed by the vCluster binary via
	// a systemd service. We write the auth config, update /etc/hosts for
	// dex resolution, patch the host-side vCluster config with apiServer
	// extraArgs, and restart the vCluster service.

	// Step 1: Write auth config and /etc/hosts inside the container.
	script := fmt.Sprintf(`set -eu
mkdir -p /etc/kubernetes/authn
printf '%%s' %q | base64 -d > %s

awk '!/dex\.dex\.svc\.cluster\.local|dex\.127\.0\.0\.1\.sslip\.io/' /etc/hosts > /etc/hosts.shoulders
printf '%%s %s\n' %q >> /etc/hosts.shoulders
printf '%%s %s\n' %q >> /etc/hosts.shoulders
cat /etc/hosts.shoulders > /etc/hosts
rm /etc/hosts.shoulders
`, authConfigEncoded, authConfigPath, dexInternalHosts, dexServiceIP, dexPublicHost, dexServiceIP)

	if out, err := containerExec(context.Background(), containerName, []string{"sh", "-c", script}); err != nil {
		return fmt.Errorf("write auth config on %s: %w\n%s", containerName, err, string(out))
	}

	// Step 2: Patch the host-side vCluster config (read-only bind mount
	// inside the container, writable on the host).
	hostConfigPath := filepath.Join(os.Getenv("HOME"), ".vcluster", "docker", "vclusters", name, "vcluster.yaml")
	if err := patchVClusterConfig(hostConfigPath); err != nil {
		return fmt.Errorf("patch vCluster config: %w", err)
	}

	// Step 3: Restart the vCluster systemd service so the apiserver
	// picks up the new --authentication-config flag.
	if out, err := containerExec(context.Background(), containerName, []string{"systemctl", "restart", "vcluster"}); err != nil {
		return fmt.Errorf("restart vcluster service on %s: %w\n%s", containerName, err, string(out))
	}

	if err := WaitForAPIServerRestart(kubeconfig, previousStartTime, apiserverRestartTimeout); err != nil {
		return fmt.Errorf("kube-apiserver did not recover after OIDC configuration: %w", err)
	}

	return nil
}

// patchVClusterConfig adds apiServer extraArgs to the vCluster config file
// on the host. The file is bind-mounted read-only into the container, so we
// edit it on the host side.
func patchVClusterConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	content := string(data)
	if strings.Contains(content, "authentication-config") {
		return nil // already patched
	}

	// Insert apiServer block right after the "    k8s:" line.
	const marker = "    k8s:\n"
	insert := marker +
		"      apiServer:\n" +
		"        extraArgs:\n" +
		"        - --authentication-config=" + authConfigPath + "\n"

	patched := strings.Replace(content, marker, insert, 1)
	if patched == content {
		return fmt.Errorf("could not find %q marker in %s", strings.TrimSpace(marker), path)
	}

	return os.WriteFile(path, []byte(patched), 0644)
}

func WaitForAPIServerRestart(kubeconfig, previousStartTime string, timeout time.Duration) error {
	clientset, err := kube.NewClientset(kubeconfig)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)

	if previousStartTime == "" {
		// No static pod to track (vind). Give the restart a moment,
		// then wait for the API server to become reachable.
		time.Sleep(3 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := clientset.Discovery().ServerVersion(); err == nil {
				return nil
			}
			time.Sleep(3 * time.Second)
		}
		return fmt.Errorf("kube-apiserver did not become ready within %s", timeout)
	}

	sawUnavailable := false
	for time.Now().Before(deadline) {
		startTimeChanged := false

		pod, err := clientset.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{LabelSelector: "component=kube-apiserver"})
		if err == nil && len(pod.Items) > 0 {
			currentStartTime := apiserverPodObservedStartTime(&pod.Items[0])
			if currentStartTime != "" && currentStartTime != previousStartTime {
				startTimeChanged = true
			}
		}

		if _, err := clientset.Discovery().ServerVersion(); err == nil {
			if sawUnavailable || startTimeChanged {
				return nil
			}
		} else {
			sawUnavailable = true
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("kube-apiserver did not become ready within %s", timeout)
}

func apiserverObservedStartTime(kubeconfig string) (string, error) {
	clientset, err := kube.NewClientset(kubeconfig)
	if err != nil {
		return "", err
	}

	pods, err := clientset.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{LabelSelector: "component=kube-apiserver"})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", nil
	}

	return apiserverPodObservedStartTime(&pods.Items[0]), nil
}

func apiserverPodObservedStartTime(pod *corev1.Pod) string {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != "kube-apiserver" {
			continue
		}
		if status.State.Running != nil {
			return status.State.Running.StartedAt.UTC().Format(time.RFC3339)
		}
	}

	if pod.Status.StartTime == nil {
		return ""
	}

	return pod.Status.StartTime.UTC().Format(time.RFC3339)
}

func dexServiceIP(kubeconfig string) (string, error) {
	clientset, err := kube.NewClientset(kubeconfig)
	if err != nil {
		return "", err
	}

	service, err := clientset.CoreV1().Services(dexNamespace).Get(context.Background(), dexServiceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get dex service: %w", err)
	}
	if service.Spec.ClusterIP == "" || service.Spec.ClusterIP == "None" {
		return defaultDexServiceIP, nil
	}

	return service.Spec.ClusterIP, nil
}

func deploymentReady(deployment *appsv1.Deployment) bool {
	if deployment.Generation > deployment.Status.ObservedGeneration {
		return false
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == "True" {
			return true
		}
	}

	return false
}
