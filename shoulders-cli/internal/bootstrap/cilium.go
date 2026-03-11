package bootstrap

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	ciliumRepoURL   = "https://helm.cilium.io/"
	ciliumChartName = "cilium"
	ciliumVersion   = "1.19.1"
)

func EnsureCilium(kubeconfigPath string) error {
	settings := helmcli.New()
	if kubeconfigPath != "" {
		settings.KubeConfig = kubeconfigPath
	}
	settings.SetNamespace("kube-system")

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "secret", log.Printf); err != nil {
		return err
	}

	chartPath, err := locateChart(settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	values := map[string]interface{}{
		"kubeProxyReplacement": true,
		"image": map[string]interface{}{
			"pullPolicy": "IfNotPresent",
		},
		"ipam": map[string]interface{}{
			"mode": "kubernetes",
		},
		"gatewayAPI": map[string]interface{}{
			"enabled": true,
			"hostNetwork": map[string]interface{}{
				"enabled": true,
			},
		},
		"envoy": map[string]interface{}{
			"securityContext": map[string]interface{}{
				"capabilities": map[string]interface{}{
					"envoy":                 []string{"NET_ADMIN", "SYS_ADMIN", "NET_BIND_SERVICE"},
					"keepCapNetBindService": true,
				},
			},
		},
	}

	if releaseExists(actionConfig, ciliumChartName) {
		upgrade := action.NewUpgrade(actionConfig)
		upgrade.Namespace = settings.Namespace()
		upgrade.Version = ciliumVersion
		if _, err = upgrade.Run(ciliumChartName, chart, values); err != nil {
			return err
		}
		return RestartCiliumWorkloads(kubeconfigPath)
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = ciliumChartName
	install.Namespace = settings.Namespace()
	install.CreateNamespace = true
	install.Version = ciliumVersion
	_, err = install.Run(chart, values)
	return err
}

func RestartCiliumWorkloads(kubeconfigPath string) error {
	clientset, err := kube.NewClientset(kubeconfigPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	restartedAt := time.Now().UTC().Format(time.RFC3339Nano)

	daemonSet, err := clientset.AppsV1().DaemonSets("kube-system").Get(ctx, "cilium", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get cilium daemonset: %w", err)
	}
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = map[string]string{}
	}
	daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
	if _, err := clientset.AppsV1().DaemonSets("kube-system").Update(ctx, daemonSet, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("restart cilium daemonset: %w", err)
	}

	deployment, err := clientset.AppsV1().Deployments("kube-system").Get(ctx, "cilium-operator", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get cilium operator deployment: %w", err)
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
	if _, err := clientset.AppsV1().Deployments("kube-system").Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("restart cilium operator deployment: %w", err)
	}

	if err := waitForCiliumDaemonSet(ctx, clientset); err != nil {
		return err
	}
	return waitForCiliumOperator(ctx, clientset)
}

func waitForCiliumDaemonSet(ctx context.Context, clientset *kubernetes.Clientset) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		daemonSet, err := clientset.AppsV1().DaemonSets("kube-system").Get(ctx, "cilium", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		desired := daemonSet.Status.DesiredNumberScheduled
		return daemonSet.Status.ObservedGeneration >= daemonSet.Generation &&
				daemonSet.Status.UpdatedNumberScheduled == desired &&
				daemonSet.Status.NumberAvailable == desired,
			nil
	})
}

func waitForCiliumOperator(ctx context.Context, clientset *kubernetes.Clientset) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		deployment, err := clientset.AppsV1().Deployments("kube-system").Get(ctx, "cilium-operator", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		replicas := int32(1)
		if deployment.Spec.Replicas != nil {
			replicas = *deployment.Spec.Replicas
		}

		return deployment.Status.ObservedGeneration >= deployment.Generation &&
				deployment.Status.UpdatedReplicas == replicas &&
				deployment.Status.AvailableReplicas == replicas,
			nil
	})
}

func locateChart(settings *helmcli.EnvSettings) (string, error) {
	chartPathOptions := &action.ChartPathOptions{
		RepoURL: ciliumRepoURL,
		Version: ciliumVersion,
	}
	return chartPathOptions.LocateChart(ciliumChartName, settings)
}

func releaseExists(cfg *action.Configuration, name string) bool {
	get := action.NewGet(cfg)
	_, err := get.Run(name)
	return err == nil
}
