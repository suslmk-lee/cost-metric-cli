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

	// If no CSP was detected, default to NHN Cloud.
	if !cspInfo.Detected {
		cspInfo.Type = CSPTypeNHN
		cspInfo.Name = getCSPName(CSPTypeNHN)
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

	// ProviderID는 노드가 어떤 클라우드에서 실행 중인지 알려주는 가장 확실한 정보
	for _, node := range nodes {
		if node.ProviderID != "" {
			// NHN Cloud ProviderID는 "nhncloud://"로 시작합니다.
			if strings.HasPrefix(node.ProviderID, "nhncloud://") {
				return CSPTypeNHN
			}
			// AWS ProviderID는 "aws://"로 시작합니다.
			if strings.HasPrefix(node.ProviderID, "aws://") {
				return CSPTypeAWS
			}
			// GCP ProviderID는 "gce://"로 시작합니다.
			if strings.HasPrefix(node.ProviderID, "gce://") {
				return CSPTypeGCP
			}
			// Azure ProviderID는 "azure://"로 시작합니다.
			if strings.HasPrefix(node.ProviderID, "azure://") {
				return CSPTypeAzure
			}
			// Naver Cloud ProviderID는 "ncloud://"로 시작합니다.
			if strings.HasPrefix(node.ProviderID, "ncloud://") {
				return CSPTypeNaver
			}
		}
	}

	// IP 대역을 통한 탐지
	for _, node := range nodes {
		// 노드의 내부 IP를 확인합니다
		ipAddress := node.Labels["kubernetes.io/hostname"]
		if ipAddress == "" {
			// 호스트명에서 IP를 추출할 수 없는 경우 다음 노드로 넘어갑니다
			continue
		}

		// NHN Cloud IP 대역 확인 (예: 133.186.x.x, 211.56.x.x)
		if strings.HasPrefix(ipAddress, "133.186.") || strings.HasPrefix(ipAddress, "211.56.") {
			return CSPTypeNHN
		}

		// Naver Cloud IP 대역 확인 (예: 101.101.x.x, 175.45.x.x)
		if strings.HasPrefix(ipAddress, "101.101.") || strings.HasPrefix(ipAddress, "175.45.") {
			return CSPTypeNaver
		}
	}

	// 노드 이름에서 CSP 관련 패턴을 찾습니다.
	for _, node := range nodes {
		nodeName := strings.ToLower(node.Name)

		// AWS EKS 노드 이름 패턴 (예: ip-10-0-1-20.ec2.internal, ip-192-168-1-100.us-west-2.compute.internal)
		if strings.Contains(nodeName, ".ec2.internal") || strings.Contains(nodeName, ".compute.internal") {
			return CSPTypeAWS
		}

		// GCP GKE 노드 이름 패턴 (예: gke-cluster-1-default-pool-12345678-abcd)
		if strings.HasPrefix(nodeName, "gke-") {
			return CSPTypeGCP
		}

		// Azure AKS 노드 이름 패턴 (예: aks-nodepool1-12345678-vmss000000)
		if strings.HasPrefix(nodeName, "aks-") {
			return CSPTypeAzure
		}

		// Naver Cloud 노드 이름 패턴 (예: nks-nodepool-xxx-yyyyy)
		if strings.HasPrefix(nodeName, "nks-") {
			return CSPTypeNaver
		}

		// NHN Cloud 노드 이름 패턴 (예: node-pool-xxx-yyyyy)
		if strings.HasPrefix(nodeName, "node-pool-") &&
			(strings.Contains(nodeName, "-toast-") || strings.Contains(nodeName, "-nhn-")) {
			return CSPTypeNHN
		}
	}

	// --- 시작: 노드 OS 이미지 및 커널 버전을 통한 탐지 로직 추가 ---
	for _, node := range nodes {
		osImage := strings.ToLower(node.OSImage)
		kernelVersion := strings.ToLower(node.KernelVersion)

		// AWS EKS는 주로 Amazon Linux 또는 특정 커널 버전을 사용합니다
		if strings.Contains(osImage, "amazon linux") || strings.Contains(osImage, "eks") {
			return CSPTypeAWS
		}

		// GCP GKE는 Container-Optimized OS (COS) 또는 Ubuntu를 사용합니다
		if strings.Contains(osImage, "container-optimized os") || strings.Contains(osImage, "cos") {
			return CSPTypeGCP
		}

		// Azure AKS는 특정 Ubuntu 버전이나 CBL-Mariner를 사용합니다
		if strings.Contains(osImage, "cbl-mariner") ||
			(strings.Contains(osImage, "ubuntu") && strings.Contains(kernelVersion, "azure")) {
			return CSPTypeAzure
		}
	}
	// --- 종료: OS 이미지 및 커널 버전 탐지 로직 ---

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

	// Azure AKS indicators
	azureIndicators := []string{
		"kubernetes.azure.com",
		"agentpool",
		"storageprofile",
		"storagetier",
		"accelerator",
	}

	// GCP GKE indicators
	gcpIndicators := []string{
		"cloud.google.com",
		"gke.io",
		"topology.gke.io",
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

	// Check for Azure AKS
	for key := range labels {
		for _, indicator := range azureIndicators {
			if strings.Contains(strings.ToLower(key), indicator) {
				return CSPTypeAzure
			}
		}
	}

	// Check for GCP GKE
	for key := range labels {
		for _, indicator := range gcpIndicators {
			if strings.Contains(strings.ToLower(key), indicator) {
				return CSPTypeGCP
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

		// AWS instance type patterns (e.g., t2.micro, m5.large)
		if strings.Contains(instanceType, "t2.") || strings.Contains(instanceType, "t3.") ||
			strings.Contains(instanceType, "m5.") || strings.Contains(instanceType, "c5.") ||
			strings.Contains(instanceType, "r5.") {
			return CSPTypeAWS
		}

		// GCP instance type patterns (e.g., e2-standard-2, n1-standard-1)
		if strings.Contains(instanceType, "e2-") || strings.Contains(instanceType, "n1-") ||
			strings.Contains(instanceType, "n2-") || strings.Contains(instanceType, "c2-") {
			return CSPTypeGCP
		}

		// Azure instance type patterns (e.g., Standard_D2s_v3, Standard_B2ms)
		if strings.HasPrefix(instanceType, "standard_") {
			return CSPTypeAzure
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
