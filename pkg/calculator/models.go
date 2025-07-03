package calculator

import (
	"time"

	"github.com/minkyulee/cost-metric-cli/pkg/csp"
	"github.com/minkyulee/cost-metric-cli/pkg/k8s"
)

// PricingInfo contains pricing information for an instance type
type PricingInfo struct {
	InstanceType string    `json:"instance_type"`
	Region       string    `json:"region"`
	HourlyPrice  float64   `json:"hourly_price"`
	Currency     string    `json:"currency"`
	CPU          int       `json:"cpu"`
	Memory       int       `json:"memory"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NodeCost represents the cost information for a single node
type NodeCost struct {
	NodeInfo    k8s.NodeInfo  `json:"node_info"`
	PricingInfo PricingInfo   `json:"pricing_info"`
	HourlyCost  float64       `json:"hourly_cost"`
	DailyCost   float64       `json:"daily_cost"`
	MonthlyCost float64       `json:"monthly_cost"`
	YearlyCost  float64       `json:"yearly_cost"`
	CustomCost  float64       `json:"custom_cost"`
	Duration    time.Duration `json:"duration"`
	// Actual usage calculation
	ActualUsageCost float64       `json:"actual_usage_cost"`
	ActualUptime    time.Duration `json:"actual_uptime"`
	UptimeHours     float64       `json:"uptime_hours"`
}

// CostBreakdown provides detailed cost breakdown information
type CostBreakdown struct {
	HourlyTotal  float64            `json:"hourly_total"`
	DailyTotal   float64            `json:"daily_total"`
	MonthlyTotal float64            `json:"monthly_total"`
	YearlyTotal  float64            `json:"yearly_total"`
	CustomTotal  float64            `json:"custom_total"`
	Duration     time.Duration      `json:"duration"`
	Currency     string             `json:"currency"`
	ByType       map[string]float64 `json:"by_type"`
	ByRegion     map[string]float64 `json:"by_region"`
	ByStatus     map[string]float64 `json:"by_status"`
	// Actual usage totals
	ActualUsageTotal float64 `json:"actual_usage_total"`
	TotalUptimeHours float64 `json:"total_uptime_hours"`
	// Kubernetes service costs
	KubernetesServiceCost KubernetesServiceCost `json:"kubernetes_service_cost"`
}

// KubernetesServiceCost represents additional Kubernetes service costs
type KubernetesServiceCost struct {
	ClusterManagementCost ServiceCost `json:"cluster_management_cost"`
	LoadBalancerCost      ServiceCost `json:"load_balancer_cost"`
	TotalServiceCost      ServiceCost `json:"total_service_cost"`
}

// ServiceCost represents cost for a specific service
type ServiceCost struct {
	HourlyCost      float64 `json:"hourly_cost"`
	DailyCost       float64 `json:"daily_cost"`
	MonthlyCost     float64 `json:"monthly_cost"`
	YearlyCost      float64 `json:"yearly_cost"`
	CustomCost      float64 `json:"custom_cost"`
	ActualUsageCost float64 `json:"actual_usage_cost"`
	Description     string  `json:"description"`
}

// CostReport contains comprehensive cost analysis results
type CostReport struct {
	ClusterName    string         `json:"cluster_name"`
	Context        string         `json:"context"`
	CSP            csp.CSPInfo    `json:"csp"`
	Nodes          []NodeCost     `json:"nodes"`
	TotalCost      CostBreakdown  `json:"total_cost"`
	Summary        ClusterSummary `json:"summary"`
	GeneratedAt    time.Time      `json:"generated_at"`
	AnalysisPeriod time.Duration  `json:"analysis_period"`
}

// ClusterSummary provides a summary of cluster resources
type ClusterSummary struct {
	TotalNodes     int            `json:"total_nodes"`
	RunningNodes   int            `json:"running_nodes"`
	NodesByType    map[string]int `json:"nodes_by_type"`
	NodesByRegion  map[string]int `json:"nodes_by_region"`
	NodesByStatus  map[string]int `json:"nodes_by_status"`
	TotalCPU       int            `json:"total_cpu"`
	TotalMemory    int64          `json:"total_memory"`
	AvgCostPerNode float64        `json:"avg_cost_per_node"`
	CostPerCPU     float64        `json:"cost_per_cpu"`
	CostPerGB      float64        `json:"cost_per_gb"`
}

// CostAnalysisRequest contains parameters for cost analysis
type CostAnalysisRequest struct {
	Context    string        `json:"context"`
	Duration   time.Duration `json:"duration"`
	Region     string        `json:"region,omitempty"`
	OutputPath string        `json:"output_path,omitempty"`
	Verbose    bool          `json:"verbose"`
}

// CostOptimization provides cost optimization suggestions
type CostOptimization struct {
	CurrentCost     float64                  `json:"current_cost"`
	OptimizedCost   float64                  `json:"optimized_cost"`
	PotentialSaving float64                  `json:"potential_saving"`
	Suggestions     []OptimizationSuggestion `json:"suggestions"`
}

// OptimizationSuggestion represents a single optimization recommendation
type OptimizationSuggestion struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Saving      float64 `json:"saving"`
	Effort      string  `json:"effort"`
}

// PricingRepository interface for retrieving pricing information
type PricingRepository interface {
	GetPricing(cspType csp.CSPType, instanceType, region string) (*PricingInfo, error)
	GetAllPricing(cspType csp.CSPType, region string) (map[string]*PricingInfo, error)
	UpdatePricing(cspType csp.CSPType, region string) error
	GetKubernetesServicePricing(cspType csp.CSPType) (*KubernetesServicePricing, error)
	GetCSPInfo(cspType csp.CSPType) (*CSPConfig, error)
	GetSupportedInstanceTypes(cspType csp.CSPType) ([]string, error)
}

// CostCalculator interface for cost calculation operations
type CostCalculator interface {
	CalculateNodeCost(nodeInfo k8s.NodeInfo, pricingInfo PricingInfo, duration time.Duration) *NodeCost
	CalculateClusterCost(cspInfo *csp.CSPInfo, nodes []k8s.NodeInfo, pricingMap map[string]*PricingInfo, duration time.Duration) (*CostReport, error)
	GenerateOptimizationSuggestions(report *CostReport) *CostOptimization
}
