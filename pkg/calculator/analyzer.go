package calculator

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/minkyulee/cost-metric-cli/pkg/csp"
	"github.com/minkyulee/cost-metric-cli/pkg/k8s"
)

// CostAnalyzer provides high-level cost analysis functionality
type CostAnalyzer struct {
	calculator  CostCalculator
	pricingRepo PricingRepository
}

// NewCostAnalyzer creates a new cost analyzer
func NewCostAnalyzer(configPath string) (*CostAnalyzer, error) {
	// Initialize pricing repository
	pricingRepo, err := NewConfigPricingRepository(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pricing repository: %w", err)
	}

	// Initialize cost calculator
	calculator := NewDefaultCostCalculator(pricingRepo)

	return &CostAnalyzer{
		calculator:  calculator,
		pricingRepo: pricingRepo,
	}, nil
}

// AnalyzeCurrentContext analyzes the cost of the current Kubernetes context
func (a *CostAnalyzer) AnalyzeCurrentContext(ctx context.Context, request CostAnalysisRequest) (*CostReport, error) {
	// Create Kubernetes client
	k8sClient, err := k8s.NewK8sClient(request.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Test connection
	if err := k8sClient.TestConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Kubernetes cluster: %w", err)
	}

	// Get context information
	contextInfo := k8sClient.GetContextInfo()

	// Get cluster nodes
	nodes, err := k8sClient.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in the cluster")
	}

	// Detect CSP
	cspInfo, err := csp.DetectCSP(contextInfo, nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to detect CSP: %w", err)
	}

	if !cspInfo.IsSupported() {
		return nil, fmt.Errorf("CSP %s is not supported yet", cspInfo.Name)
	}

	// Get pricing information for all instance types
	region := request.Region
	if region == "" {
		region = cspInfo.Region
	}

	pricingMap, err := a.pricingRepo.GetAllPricing(cspInfo.Type, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing information: %w", err)
	}

	// Calculate cluster cost
	report, err := a.calculator.CalculateClusterCost(cspInfo, nodes, pricingMap, request.Duration)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cluster cost: %w", err)
	}

	// Set additional report information
	report.ClusterName = contextInfo.Cluster
	report.Context = contextInfo.Name
	report.CSP = *cspInfo

	return report, nil
}

// GetPricingInfo retrieves pricing information for a specific CSP and region
func (a *CostAnalyzer) GetPricingInfo(cspType csp.CSPType, region string) (map[string]*PricingInfo, error) {
	return a.pricingRepo.GetAllPricing(cspType, region)
}

// GenerateOptimizationReport generates cost optimization suggestions
func (a *CostAnalyzer) GenerateOptimizationReport(report *CostReport) *CostOptimization {
	return a.calculator.GenerateOptimizationSuggestions(report)
}

// GetSupportedCSPs returns list of supported CSPs with their configuration
func (a *CostAnalyzer) GetSupportedCSPs() ([]CSPConfig, error) {
	supportedTypes := csp.GetSupportedCSPs()
	var configs []CSPConfig

	for _, cspType := range supportedTypes {
		config, err := a.pricingRepo.GetCSPInfo(cspType)
		if err != nil {
			fmt.Printf("Warning: could not retrieve config for CSP %s: %v\n", cspType, err)
			continue
		}
		configs = append(configs, *config)
	}

	return configs, nil
}

// ValidateContext validates if the given context exists and is accessible
func (a *CostAnalyzer) ValidateContext(ctx context.Context, contextName string) error {
	k8sClient, err := k8s.NewK8sClient(contextName)
	if err != nil {
		return fmt.Errorf("invalid context %s: %w", contextName, err)
	}

	return k8sClient.TestConnection(ctx)
}

// ListAvailableContexts returns all available Kubernetes contexts
func (a *CostAnalyzer) ListAvailableContexts() ([]string, error) {
	return k8s.ListContexts()
}

// GetCurrentContext returns the current Kubernetes context
func (a *CostAnalyzer) GetCurrentContext() (string, error) {
	return k8s.GetCurrentContext()
}

// NewDefaultAnalyzer creates a cost analyzer with default configuration
func NewDefaultAnalyzer() (*CostAnalyzer, error) {
	// Use default config path
	configPath := filepath.Join("configs", "pricing.yaml")
	return NewCostAnalyzer(configPath)
}

// AnalysisOptions provides options for cost analysis
type AnalysisOptions struct {
	Context    string
	Duration   time.Duration
	Region     string
	Verbose    bool
	OutputPath string
	Format     string
}

// QuickAnalysis performs a quick cost analysis with minimal configuration
func (a *CostAnalyzer) QuickAnalysis(ctx context.Context, options AnalysisOptions) (*CostReport, error) {
	// Set default duration if not specified
	duration := options.Duration
	if duration == 0 {
		duration = time.Hour
	}

	// Create analysis request
	request := CostAnalysisRequest{
		Context:    options.Context,
		Duration:   duration,
		Region:     options.Region,
		OutputPath: options.OutputPath,
		Verbose:    options.Verbose,
	}

	return a.AnalyzeCurrentContext(ctx, request)
}

// GetClusterSummary returns a summary of cluster resources without cost calculation
func (a *CostAnalyzer) GetClusterSummary(ctx context.Context, contextName string) (*k8s.ClusterSummary, error) {
	k8sClient, err := k8s.NewK8sClient(contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return k8sClient.GetClusterSummary(ctx)
}
