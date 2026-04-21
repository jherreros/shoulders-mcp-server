package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var platformNamespaces = []string{
	"cnpg-system",
	"crossplane-system",
	"dex",
	"falco",
	"headlamp",
	"kafka",
	"kyverno",
	"observability",
	"policy-reporter",
	"strimzi",
	"trivy-system",
}

func CleanupKyvernoWebhooks(kubeconfigPath string) error {
	clientset, err := kube.NewClientset(kubeconfigPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	validating, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list validating webhook configurations: %w", err)
	}
	for _, item := range validating.Items {
		if strings.HasPrefix(item.Name, "kyverno-") {
			if err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, item.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("delete validating webhook configuration %q: %w", item.Name, err)
			}
		}
	}

	mutating, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list mutating webhook configurations: %w", err)
	}
	for _, item := range mutating.Items {
		if strings.HasPrefix(item.Name, "kyverno-") {
			if err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, item.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("delete mutating webhook configuration %q: %w", item.Name, err)
			}
		}
	}

	return nil
}

func WaitForNamespacesDeleted(ctx context.Context, kubeconfigPath string) error {
	clientset, err := kube.NewClientset(kubeconfigPath)
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		for _, namespace := range platformNamespaces {
			if _, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); err == nil {
				return false, nil
			} else if !apierrors.IsNotFound(err) {
				return false, fmt.Errorf("get namespace %q: %w", namespace, err)
			}
		}
		return true, nil
	})
}
