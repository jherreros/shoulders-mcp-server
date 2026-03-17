package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/jherreros/shoulders/shoulders-cli/internal/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var httpRouteGVR = schema.GroupVersionResource{
	Group:    "gateway.networking.k8s.io",
	Version:  "v1",
	Resource: "httproutes",
}

func WaitForHTTPRouteResolved(kubeconfig, namespace, name string, timeout time.Duration) error {
	dynamicClient, err := kube.NewDynamicClient(kubeconfig)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		route, err := dynamicClient.Resource(httpRouteGVR).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err == nil && httpRouteResolved(route) {
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("httproute %s/%s did not resolve within %s", namespace, name, timeout)
}

func httpRouteResolved(route *unstructured.Unstructured) bool {
	parents, found, err := unstructured.NestedSlice(route.Object, "status", "parents")
	if err != nil || !found {
		return false
	}

	for _, parent := range parents {
		parentMap, ok := parent.(map[string]interface{})
		if !ok {
			continue
		}
		conditions, found, err := unstructured.NestedSlice(parentMap, "conditions")
		if err != nil || !found {
			continue
		}

		accepted := false
		resolved := false
		for _, condition := range conditions {
			conditionMap, ok := condition.(map[string]interface{})
			if !ok {
				continue
			}
			conditionType, _, _ := unstructured.NestedString(conditionMap, "type")
			status, _, _ := unstructured.NestedString(conditionMap, "status")
			switch conditionType {
			case "Accepted":
				accepted = status == "True"
			case "ResolvedRefs":
				resolved = status == "True"
			}
		}

		if accepted && resolved {
			return true
		}
	}

	return false
}