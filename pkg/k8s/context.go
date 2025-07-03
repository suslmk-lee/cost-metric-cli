package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ContextInfo struct {
	Name      string
	Cluster   string
	Namespace string
	User      string
	Server    string
}

type K8sClient struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
	context   *ContextInfo
}

// NewK8sClient creates a new Kubernetes client with the specified context
func NewK8sClient(contextName string) (*K8sClient, error) {
	config, err := getKubeConfig(contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	contextInfo, err := getCurrentContextInfo(contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to get context info: %w", err)
	}

	return &K8sClient{
		clientset: clientset,
		config:    config,
		context:   contextInfo,
	}, nil
}

// getKubeConfig loads kubeconfig and returns the client configuration
func getKubeConfig(contextName string) (*rest.Config, error) {
	// Try in-cluster config first
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	kubeconfig := getKubeconfigPath()
	
	configLoadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeconfig,
	}
	
	configOverrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		configOverrides.CurrentContext = contextName
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		configOverrides,
	).ClientConfig()
	
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return config, nil
}

// getKubeconfigPath returns the path to the kubeconfig file
func getKubeconfigPath() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	
	return ""
}

// getCurrentContextInfo retrieves information about the current context
func getCurrentContextInfo(contextName string) (*ContextInfo, error) {
	kubeconfig := getKubeconfigPath()
	
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig file: %w", err)
	}

	currentContext := contextName
	if currentContext == "" {
		currentContext = config.CurrentContext
	}

	context, exists := config.Contexts[currentContext]
	if !exists {
		return nil, fmt.Errorf("context %s not found in kubeconfig", currentContext)
	}

	cluster, exists := config.Clusters[context.Cluster]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found in kubeconfig", context.Cluster)
	}

	return &ContextInfo{
		Name:      currentContext,
		Cluster:   context.Cluster,
		Namespace: context.Namespace,
		User:      context.AuthInfo,
		Server:    cluster.Server,
	}, nil
}

// ListContexts returns all available contexts from kubeconfig
func ListContexts() ([]string, error) {
	kubeconfig := getKubeconfigPath()
	
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig file: %w", err)
	}

	var contexts []string
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}

	return contexts, nil
}

// GetCurrentContext returns the current context name
func GetCurrentContext() (string, error) {
	kubeconfig := getKubeconfigPath()
	
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig file: %w", err)
	}

	return config.CurrentContext, nil
}

// GetContextInfo returns information about the specified context
func (c *K8sClient) GetContextInfo() *ContextInfo {
	return c.context
}

// GetClientset returns the Kubernetes clientset
func (c *K8sClient) GetClientset() *kubernetes.Clientset {
	return c.clientset
}

// TestConnection tests the connection to the Kubernetes cluster
func (c *K8sClient) TestConnection(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	return nil
}