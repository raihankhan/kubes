package kube

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const podTimeout = 15 * time.Second

// Pod holds the essential display fields for a Kubernetes pod.
type Pod struct {
	Name     string
	Ready    string // "2/3"
	Status   string // phase or granular condition
	Restarts int
	Age      string
}

// GetPodsForNamespace returns pods in the given namespace for the specified context.
// contextName must be the name as it appears inside the kubeconfig file at kubeconfigPath.
func GetPodsForNamespace(contextName, namespace, kubeconfigPath string) ([]Pod, error) {
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
	config.Timeout = podTimeout

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), podTimeout)
	defer cancel()

	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods in %s: %w", namespace, err)
	}

	pods := make([]Pod, 0, len(podList.Items))
	for _, p := range podList.Items {
		total := len(p.Spec.Containers)
		ready := 0
		restarts := 0
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += int(cs.RestartCount)
		}

		// Prefer a granular status reason over the broad phase.
		status := string(p.Status.Phase)
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				status = cs.State.Waiting.Reason
				break
			}
			if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				status = cs.State.Terminated.Reason
				break
			}
		}

		pods = append(pods, Pod{
			Name:     p.Name,
			Ready:    fmt.Sprintf("%d/%d", ready, total),
			Status:   status,
			Restarts: restarts,
			Age:      formatAge(p.CreationTimestamp.Time),
		})
	}
	return pods, nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
