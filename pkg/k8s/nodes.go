package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeInfo struct {
	Name         string            `json:"name"`
	InstanceType string            `json:"instance_type"`
	Region       string            `json:"region"`
	Zone         string            `json:"zone"`
	Status       string            `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	Labels       map[string]string `json:"labels"`
	CPU          string            `json:"cpu"`
	Memory       string            `json:"memory"`
	OSImage      string            `json:"os_image"`
	KernelVersion string           `json:"kernel_version"`
}

// GetNodes retrieves all nodes from the Kubernetes cluster
func (c *K8sClient) GetNodes(ctx context.Context) ([]NodeInfo, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var nodeInfos []NodeInfo
	for _, node := range nodes.Items {
		nodeInfo := convertToNodeInfo(&node)
		nodeInfos = append(nodeInfos, nodeInfo)
	}

	return nodeInfos, nil
}

// GetNode retrieves a specific node by name
func (c *K8sClient) GetNode(ctx context.Context, nodeName string) (*NodeInfo, error) {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	nodeInfo := convertToNodeInfo(node)
	return &nodeInfo, nil
}

// convertToNodeInfo converts a Kubernetes Node object to NodeInfo
func convertToNodeInfo(node *corev1.Node) NodeInfo {
	nodeInfo := NodeInfo{
		Name:      node.Name,
		CreatedAt: node.CreationTimestamp.Time,
		Labels:    node.Labels,
		Status:    getNodeStatus(node),
	}

	// Extract instance type from labels (varies by CSP)
	nodeInfo.InstanceType = extractInstanceType(node.Labels)
	
	// Extract region and zone information
	nodeInfo.Region = extractRegion(node.Labels)
	nodeInfo.Zone = extractZone(node.Labels)

	// Extract resource information
	if cpu := node.Status.Capacity.Cpu(); cpu != nil {
		nodeInfo.CPU = cpu.String()
	}
	if memory := node.Status.Capacity.Memory(); memory != nil {
		nodeInfo.Memory = memory.String()
	}

	// Extract system information
	if len(node.Status.NodeInfo.OSImage) > 0 {
		nodeInfo.OSImage = node.Status.NodeInfo.OSImage
	}
	if len(node.Status.NodeInfo.KernelVersion) > 0 {
		nodeInfo.KernelVersion = node.Status.NodeInfo.KernelVersion
	}

	return nodeInfo
}

// getNodeStatus determines the overall status of a node
func getNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

// extractInstanceType extracts instance type from node labels
func extractInstanceType(labels map[string]string) string {
	// Try different label keys used by various CSPs
	possibleKeys := []string{
		"node.kubernetes.io/instance-type",
		"beta.kubernetes.io/instance-type",
		"kubernetes.io/instance-type",
		"ncloud.com/instance-type",
		"toast.com/instance-type",
	}

	for _, key := range possibleKeys {
		if value, exists := labels[key]; exists {
			return value
		}
	}

	return "unknown"
}

// extractRegion extracts region information from node labels
func extractRegion(labels map[string]string) string {
	possibleKeys := []string{
		"topology.kubernetes.io/region",
		"failure-domain.beta.kubernetes.io/region",
		"kubernetes.io/region",
		"ncloud.com/region",
		"toast.com/region",
	}

	for _, key := range possibleKeys {
		if value, exists := labels[key]; exists {
			return value
		}
	}

	return "unknown"
}

// extractZone extracts zone information from node labels
func extractZone(labels map[string]string) string {
	possibleKeys := []string{
		"topology.kubernetes.io/zone",
		"failure-domain.beta.kubernetes.io/zone",
		"kubernetes.io/zone",
		"ncloud.com/zone",
		"toast.com/zone",
	}

	for _, key := range possibleKeys {
		if value, exists := labels[key]; exists {
			return value
		}
	}

	return "unknown"
}

// GetNodesByInstanceType groups nodes by their instance type
func (c *K8sClient) GetNodesByInstanceType(ctx context.Context) (map[string][]NodeInfo, error) {
	nodes, err := c.GetNodes(ctx)
	if err != nil {
		return nil, err
	}

	nodesByType := make(map[string][]NodeInfo)
	for _, node := range nodes {
		instanceType := node.InstanceType
		nodesByType[instanceType] = append(nodesByType[instanceType], node)
	}

	return nodesByType, nil
}

// GetNodesByRegion groups nodes by their region
func (c *K8sClient) GetNodesByRegion(ctx context.Context) (map[string][]NodeInfo, error) {
	nodes, err := c.GetNodes(ctx)
	if err != nil {
		return nil, err
	}

	nodesByRegion := make(map[string][]NodeInfo)
	for _, node := range nodes {
		region := node.Region
		nodesByRegion[region] = append(nodesByRegion[region], node)
	}

	return nodesByRegion, nil
}

// GetClusterSummary returns a summary of the cluster's nodes
func (c *K8sClient) GetClusterSummary(ctx context.Context) (*ClusterSummary, error) {
	nodes, err := c.GetNodes(ctx)
	if err != nil {
		return nil, err
	}

	summary := &ClusterSummary{
		TotalNodes:      len(nodes),
		NodesByType:     make(map[string]int),
		NodesByRegion:   make(map[string]int),
		NodesByStatus:   make(map[string]int),
	}

	for _, node := range nodes {
		summary.NodesByType[node.InstanceType]++
		summary.NodesByRegion[node.Region]++
		summary.NodesByStatus[node.Status]++
	}

	return summary, nil
}

type ClusterSummary struct {
	TotalNodes    int            `json:"total_nodes"`
	NodesByType   map[string]int `json:"nodes_by_type"`
	NodesByRegion map[string]int `json:"nodes_by_region"`
	NodesByStatus map[string]int `json:"nodes_by_status"`
}