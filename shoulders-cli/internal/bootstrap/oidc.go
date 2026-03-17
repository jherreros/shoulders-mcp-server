package bootstrap

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kind/pkg/cluster/constants"
)

const (
	authConfigPath          = "/etc/kubernetes/authn/authentication-config.yaml"
	apiserverManifest       = "/etc/kubernetes/manifests/kube-apiserver.yaml"
	apiserverManifestBackup = "/etc/kubernetes/kube-apiserver.yaml.pre-oidc"
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
	provider, err := newProvider()
	if err != nil {
		return err
	}

	dexServiceIP, err := dexServiceIP(kubeconfig)
	if err != nil {
		return err
	}

	nodes, err := provider.ListNodes(name)
	if err != nil {
		return fmt.Errorf("list kind nodes: %w", err)
	}

	for _, node := range nodes {
		role, err := node.Role()
		if err != nil {
			return fmt.Errorf("get node role for %s: %w", node.String(), err)
		}
		if role != constants.ControlPlaneNodeRoleValue {
			continue
		}

		previousStartTime, err := apiserverObservedStartTime(kubeconfig)
		if err != nil {
			return fmt.Errorf("get kube-apiserver start time: %w", err)
		}

		authConfigEncoded := base64.StdEncoding.EncodeToString(authConfig)

		script := fmt.Sprintf(`set -eu
mkdir -p /etc/kubernetes/authn
printf '%%s' %q | base64 -d > %s

awk '!/dex\.dex\.svc\.cluster\.local|dex\.127\.0\.0\.1\.sslip\.io/' /etc/hosts > /etc/hosts.shoulders
printf '%%s %s\n' %q >> /etc/hosts.shoulders
printf '%%s %s\n' %q >> /etc/hosts.shoulders
cat /etc/hosts.shoulders > /etc/hosts
rm /etc/hosts.shoulders

cp %s %s
tmp=$(mktemp)
cp %s "$tmp"

perl -0pi -e 's@\n\s*-\s+--authentication-config=%s@@g; s@\n\s*- mountPath: /etc/kubernetes/authn\n\s*name: k8s-authn\n\s*readOnly: true@@g; s@\n\s*- hostPath:\n\s*path: /etc/kubernetes/authn\n\s*type: DirectoryOrCreate\n\s*name: k8s-authn@@g' "$tmp"
perl -0pi -e 's@(\n\s*- kube-apiserver\n)@$1    - --authentication-config=%s\n@' "$tmp"
perl -0pi -e 's@(\n\s*- mountPath: /etc/kubernetes/pki\n\s*name: k8s-certs\n\s*readOnly: true\n)@$1    - mountPath: /etc/kubernetes/authn\n      name: k8s-authn\n      readOnly: true\n@' "$tmp"
perl -0pi -e 's@(\n\s*- hostPath:\n\s*path: /etc/kubernetes/pki\n\s*type: DirectoryOrCreate\n\s*name: k8s-certs\n)@$1  - hostPath:\n      path: /etc/kubernetes/authn\n      type: DirectoryOrCreate\n    name: k8s-authn\n@' "$tmp"

mv "$tmp" %s
`, authConfigEncoded, authConfigPath, dexInternalHosts, dexServiceIP, dexPublicHost, dexServiceIP, apiserverManifest, apiserverManifestBackup, apiserverManifest, authConfigPath, authConfigPath, apiserverManifest)

		if err := node.Command("sh", "-c", script).Run(); err != nil {
			return fmt.Errorf("write authentication config on %s: %w", node.String(), err)
		}

		if err := WaitForAPIServerRestart(kubeconfig, previousStartTime, apiserverRestartTimeout); err != nil {
			rollback := "cp " + apiserverManifestBackup + " " + apiserverManifest
			if rollbackErr := node.Command("sh", "-c", rollback).Run(); rollbackErr != nil {
				return fmt.Errorf("kube-apiserver did not recover after OIDC configuration (%w) and rollback failed on %s: %w", err, node.String(), rollbackErr)
			}
			rollbackStartTime, startErr := apiserverObservedStartTime(kubeconfig)
			if startErr != nil {
				rollbackStartTime = ""
			}
			if rollbackWaitErr := WaitForAPIServerRestart(kubeconfig, rollbackStartTime, apiserverRestartTimeout); rollbackWaitErr != nil {
				return fmt.Errorf("kube-apiserver did not recover after OIDC configuration (%w) and rollback also failed: %w", err, rollbackWaitErr)
			}
			return fmt.Errorf("kube-apiserver did not recover after OIDC configuration, rolled back manifest: %w", err)
		}

		return nil
	}

	return fmt.Errorf("no control-plane node found for cluster %q", name)
}

func WaitForAPIServerRestart(kubeconfig, previousStartTime string, timeout time.Duration) error {
	clientset, err := kube.NewClientset(kubeconfig)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	sawUnavailable := false
	for time.Now().Before(deadline) {
		startTimeChanged := false

		if previousStartTime != "" {
			pod, err := clientset.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{LabelSelector: "component=kube-apiserver"})
			if err == nil && len(pod.Items) > 0 {
				currentStartTime := apiserverPodObservedStartTime(&pod.Items[0])
				if currentStartTime != "" && currentStartTime != previousStartTime {
					startTimeChanged = true
				}
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
