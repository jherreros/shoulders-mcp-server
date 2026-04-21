package kube

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func ApplyManifest(ctx context.Context, kubeconfigPath string, content []byte, defaultNamespace string) error {
	return processManifest(ctx, kubeconfigPath, content, defaultNamespace, Apply)
}

func DeleteManifest(ctx context.Context, kubeconfigPath string, content []byte, defaultNamespace string) error {
	return processManifest(ctx, kubeconfigPath, content, defaultNamespace, func(ctx context.Context, client dynamic.Interface, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured) error {
		return Delete(ctx, client, gvr, namespace, obj.GetName())
	})
}

func processManifest(ctx context.Context, kubeconfigPath string, content []byte, defaultNamespace string, action func(context.Context, dynamic.Interface, schema.GroupVersionResource, string, *unstructured.Unstructured) error) error {
	restConfig, err := NewRestConfig(kubeconfigPath)
	if err != nil {
		return err
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	resources := make([]*unstructured.Unstructured, 0)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)
	for {
		var raw unstructured.Unstructured
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if raw.Object == nil {
			continue
		}
		resources = append(resources, raw.DeepCopy())
	}

	for _, raw := range resources {
		gvk := raw.GroupVersionKind()
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		namespace := raw.GetNamespace()
		if namespace == "" && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			namespace = defaultNamespace
			raw.SetNamespace(namespace)
		}
		if namespace == "" && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			return fmt.Errorf("missing namespace for %s/%s", gvk.Kind, raw.GetName())
		}

		if err := action(ctx, dynamicClient, mapping.Resource, namespace, raw); err != nil {
			return err
		}
	}
	return nil
}

func NewDiscoveryClient(kubeconfigPath string) (*discovery.DiscoveryClient, error) {
	restConfig, err := NewRestConfig(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(restConfig)
}
