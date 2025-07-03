package csp

import (
	"fmt"
	"strings"

	"github.com/minkyulee/cost-metric-cli/pkg/k8s"
)

// CSPType represents the type of Cloud Service Provider
type CSPType string

const (
	CSPTypeNaver   CSPType = "naver"
	CSPTypeNHN     CSPType = "nhn"
	CSPTypeAWS     CSPType = "aws"
	CSPTypeAzure   CSPType = "azure"
	CSPTypeGCP     CSPType = "gcp"
	CSPTypeUnknown CSPType = "unknown"
)

// CSPInfo contains information about the detected CSP
type CSPInfo struct {
	Type     CSPType `json:"type"`
	Name     string  `json:"name"`
	Region   string  `json:"region"`
	Detected bool    `json:"detected"`
}

// DetectCSP detects the CSP type based on Kubernetes context and cluster information
func DetectCSP(contextInfo *k8s.ContextInfo, nodes []k8s.NodeInfo) (*CSPInfo, error) {
	if contextInfo == nil {
		return nil, fmt.Errorf("context info is required")
	}

	cspInfo := &CSPInfo{
		Type:     CSPTypeUnknown,
		Detected: false,
	}

	// Try to detect CSP from server URL
	if cspType := detectFromServerURL(contextInfo.Server); cspType != CSPTypeUnknown {
		cspInfo.Type = cspType
		cspInfo.Name = getCSPName(cspType)
		cspInfo.Detected = true
	}

	// Try to detect CSP from node labels if server URL detection failed
	if !cspInfo.Detected && len(nodes) > 0 {
		if cspType := detectFromNodeLabels(nodes); cspType != CSPTypeUnknown {
			cspInfo.Type = cspType
			cspInfo.Name = getCSPName(cspType)
			cspInfo.Detected = true
		}
	}

	// Extract region information
	if len(nodes) > 0 {
		cspInfo.Region = extractRegionFromNodes(nodes)
	}

	return cspInfo, nil
}

// detectFromServerURL attempts to detect CSP from the Kubernetes API server URL
func detectFromServerURL(serverURL string) CSPType {
	serverURL = strings.ToLower(serverURL)

	patterns := map[CSPType][]string{
		CSPTypeNaver: {
			"ntruss.com",
			"ncloud.com",
			"gov-ntruss.com",
		},
		CSPTypeNHN: {
			"toast.com",
			"nhncloud.com",
			"gov-toast.com",
		},
		CSPTypeAWS: {
			"eks.amazonaws.com",
			"amazon.com",
			"aws.com",
		},
		CSPTypeAzure: {
			"azmk8s.io",
			"azure.com",
			"microsoft.com",
		},
		CSPTypeGCP: {
			"container.googleapis.com",
			"gke.googleapis.com",
			"google.com",
		},
	}

	for cspType, urlPatterns := range patterns {
		for _, pattern := range urlPatterns {
			if strings.Contains(serverURL, pattern) {
				return cspType
			}
		}
	}

	return CSPTypeUnknown
}

// detectFromNodeLabels attempts to detect CSP from node labels
func detectFromNodeLabels(nodes []k8s.NodeInfo) CSPType {
	if len(nodes) == 0 {
		return CSPTypeUnknown
	}

	// Check the first node's labels for CSP indicators
	labels := nodes[0].Labels

	// Naver Cloud Platform indicators
	naverIndicators := []string{
		"ncloud.com",
		"ntruss.com",
		"node-role.ncloud.com",
	}

	// NHN Cloud indicators
	nhnIndicators := []string{
		"toast.com",
		"nhncloud.com",
		"node-role.toast.com",
	}

	// AWS EKS indicators
	awsIndicators := []string{
		"eks.amazonaws.com",
		"node-role.kubernetes.io/worker",
		"node.kubernetes.io/instance-type",
	}

	// Check for Naver Cloud
	for key := range labels {
		for _, indicator := range naverIndicators {
			if strings.Contains(strings.ToLower(key), indicator) {
				return CSPTypeNaver
			}
		}
	}

	// Check for NHN Cloud
	for key := range labels {
		for _, indicator := range nhnIndicators {
			if strings.Contains(strings.ToLower(key), indicator) {
				return CSPTypeNHN
			}
		}
	}

	// Check for AWS EKS
	for key := range labels {
		for _, indicator := range awsIndicators {
			if strings.Contains(strings.ToLower(key), indicator) {
				return CSPTypeAWS
			}
		}
	}

	// Check for specific instance type patterns
	for _, node := range nodes {
		instanceType := strings.ToLower(node.InstanceType)
		
		// Naver Cloud instance type patterns (e.g., s-1vcpu-1gb, c-2vcpu-4gb)
		if strings.Contains(instanceType, "vcpu") && strings.Contains(instanceType, "gb") {
			return CSPTypeNaver
		}
		
		// NHN Cloud instance type patterns (e.g., m1.tiny, c1.small)
		if strings.Contains(instanceType, "m1.") || strings.Contains(instanceType, "c1.") || 
		   strings.Contains(instanceType, "r1.") || strings.Contains(instanceType, "t1.") {
			return CSPTypeNHN
		}
	}

	return CSPTypeUnknown
}

// extractRegionFromNodes extracts region information from nodes
func extractRegionFromNodes(nodes []k8s.NodeInfo) string {
	if len(nodes) == 0 {
		return "unknown"
	}

	// Return the region of the first node
	if nodes[0].Region != "unknown" && nodes[0].Region != "" {
		return nodes[0].Region
	}

	return "unknown"
}

// getCSPName returns the display name for the CSP type
func getCSPName(cspType CSPType) string {
	names := map[CSPType]string{
		CSPTypeNaver:   "Naver Cloud Platform",
		CSPTypeNHN:     "NHN Cloud",
		CSPTypeAWS:     "Amazon Web Services",
		CSPTypeAzure:   "Microsoft Azure",
		CSPTypeGCP:     "Google Cloud Platform",
		CSPTypeUnknown: "Unknown",
	}

	if name, exists := names[cspType]; exists {
		return name
	}

	return "Unknown"
}

// IsSupported checks if the CSP type is supported for cost analysis
func (c *CSPInfo) IsSupported() bool {
	supportedCSPs := []CSPType{
		CSPTypeNaver,
		CSPTypeNHN,
	}

	for _, supported := range supportedCSPs {
		if c.Type == supported {
			return true
		}
	}

	return false
}

// GetSupportedCSPs returns a list of supported CSP types
func GetSupportedCSPs() []CSPType {
	return []CSPType{
		CSPTypeNaver,
		CSPTypeNHN,
	}
}

// String returns the string representation of CSPType
func (c CSPType) String() string {
	return string(c)
}