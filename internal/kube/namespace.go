package kube

import (
	"context"
	"fmt"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const namespaceTimeout = 10 * time.Second

// GetNamespaces fetches all namespaces for the currently active context.
// It uses the default kubeconfig and respects a 10-second timeout.
func GetNamespaces() ([]string, error) {
	kubeconfigPath := DefaultKubeConfigPath()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	// Apply conservative timeouts to avoid hanging on unreachable clusters.
	config.Timeout = namespaceTimeout

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), namespaceTimeout)
	defer cancel()

	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}

	namespaces := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	sort.Strings(namespaces)
	return namespaces, nil
}

// GetNamespacesForContext fetches namespaces for a specific context by temporarily
// building a rest.Config for that context.
func GetNamespacesForContext(contextName, kubeconfigPath string) ([]string, error) {
	if kubeconfigPath == "" {
		kubeconfigPath = DefaultKubeConfigPath()
	}

	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	}
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		configOverrides,
	)

	config, err := loader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("building config for context %s: %w", contextName, err)
	}
	config.Timeout = namespaceTimeout

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), namespaceTimeout)
	defer cancel()

	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces for context %s: %w", contextName, err)
	}

	namespaces := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	sort.Strings(namespaces)
	return namespaces, nil
}
