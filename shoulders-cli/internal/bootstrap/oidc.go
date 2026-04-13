package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	appsv1 "k8s.io/api/apps/v1"
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
