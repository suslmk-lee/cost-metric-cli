package calculator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/minkyulee/cost-metric-cli/pkg/csp"
	"github.com/minkyulee/cost-metric-cli/pkg/k8s"
)

// DefaultCostCalculator implements the CostCalculator interface
type DefaultCostCalculator struct {
	pricingRepo PricingRepository
}

// NewDefaultCostCalculator creates a new cost calculator
func NewDefaultCostCalculator(pricingRepo PricingRepository) *DefaultCostCalculator {
	return &DefaultCostCalculator{
		pricingRepo: pricingRepo,
	}
}

// CalculateNodeCost calculates the cost for a single node
func (c *DefaultCostCalculator) CalculateNodeCost(nodeInfo k8s.NodeInfo, pricingInfo PricingInfo, duration time.Duration) *NodeCost {
	nodeCost := &NodeCost{
		NodeInfo:    nodeInfo,
		PricingInfo: pricingInfo,
		Duration:    duration,
	}

	// Calculate costs for different time periods
	nodeCost.HourlyCost = pricingInfo.HourlyPrice
	nodeCost.DailyCost = pricingInfo.HourlyPrice * 24
	nodeCost.MonthlyCost = pricingInfo.HourlyPrice * 24 * 30 // Approximate month
	nodeCost.YearlyCost = pricingInfo.HourlyPrice * 24 * 365

	// Calculate custom duration cost
	hours := duration.Hours()
	nodeCost.CustomCost = pricingInfo.HourlyPrice * hours

	// Calculate actual usage cost from node creation time
	now := time.Now()
	actualUptime := now.Sub(nodeInfo.CreatedAt)
	nodeCost.ActualUptime = actualUptime
	nodeCost.UptimeHours = actualUptime.Hours()
	nodeCost.ActualUsageCost = pricingInfo.HourlyPrice * nodeCost.UptimeHours

	return nodeCost
}

// CalculateClusterCost calculates the total cost for all nodes in a cluster
func (c *DefaultCostCalculator) CalculateClusterCost(nodes []k8s.NodeInfo, pricingMap map[string]*PricingInfo, duration time.Duration) (*CostReport, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes provided for cost calculation")
	}

	report := &CostReport{
		GeneratedAt:    time.Now(),
		AnalysisPeriod: duration,
		Nodes:          make([]NodeCost, 0, len(nodes)),
	}

	// Initialize cost breakdown
	costBreakdown := &CostBreakdown{
		Duration: duration,
		ByType:   make(map[string]float64),
		ByRegion: make(map[string]float64),
		ByStatus: make(map[string]float64),
	}

	// Initialize summary
	summary := &ClusterSummary{
		NodesByType:   make(map[string]int),
		NodesByRegion: make(map[string]int),
		NodesByStatus: make(map[string]int),
	}

	var totalNodes, runningNodes int
	var totalCPU int
	var totalMemory int64

	// Calculate cost for each node
	for _, node := range nodes {
		// Get pricing information
		pricingInfo, exists := pricingMap[node.InstanceType]
		if !exists {
			// Try to get pricing from repository
			var err error
			pricingInfo, err = c.pricingRepo.GetPricing(csp.CSPTypeNaver, node.InstanceType, node.Region)
			if err != nil {
				// Use default pricing if not found
				pricingInfo = &PricingInfo{
					InstanceType: node.InstanceType,
					Region:       node.Region,
					HourlyPrice:  0.0,
					Currency:     "KRW",
					UpdatedAt:    time.Now(),
				}
			}
		}

		// Calculate node cost
		nodeCost := c.CalculateNodeCost(node, *pricingInfo, duration)
		report.Nodes = append(report.Nodes, *nodeCost)

		// Update cost breakdown
		costBreakdown.HourlyTotal += nodeCost.HourlyCost
		costBreakdown.DailyTotal += nodeCost.DailyCost
		costBreakdown.MonthlyTotal += nodeCost.MonthlyCost
		costBreakdown.YearlyTotal += nodeCost.YearlyCost
		costBreakdown.CustomTotal += nodeCost.CustomCost

		// Update actual usage totals
		costBreakdown.ActualUsageTotal += nodeCost.ActualUsageCost
		costBreakdown.TotalUptimeHours += nodeCost.UptimeHours

		// Group by instance type
		costBreakdown.ByType[node.InstanceType] += nodeCost.CustomCost

		// Group by region
		costBreakdown.ByRegion[node.Region] += nodeCost.CustomCost

		// Group by status
		costBreakdown.ByStatus[node.Status] += nodeCost.CustomCost

		// Update summary
		totalNodes++
		if node.Status == "Ready" {
			runningNodes++
		}

		summary.NodesByType[node.InstanceType]++
		summary.NodesByRegion[node.Region]++
		summary.NodesByStatus[node.Status]++

		// Parse CPU and memory
		if cpu := parseCPU(node.CPU); cpu > 0 {
			totalCPU += cpu
		}
		if memory := parseMemory(node.Memory); memory > 0 {
			totalMemory += memory
		}
	}

	// Set currency from first node's pricing info
	if len(report.Nodes) > 0 {
		costBreakdown.Currency = report.Nodes[0].PricingInfo.Currency
	}

	// Calculate average uptime for Kubernetes service costs
	var avgUptime time.Duration
	if totalNodes > 0 {
		avgUptime = time.Duration(costBreakdown.TotalUptimeHours * float64(time.Hour) / float64(totalNodes))
	}

	// Calculate Kubernetes service costs
	if err := c.calculateKubernetesServiceCosts(costBreakdown, duration, avgUptime); err != nil {
		// Log error but continue with node costs
		fmt.Printf("Warning: Failed to calculate Kubernetes service costs: %v\n", err)
	}

	// Finalize summary
	summary.TotalNodes = totalNodes
	summary.RunningNodes = runningNodes
	summary.TotalCPU = totalCPU
	summary.TotalMemory = totalMemory

	if totalNodes > 0 {
		summary.AvgCostPerNode = costBreakdown.CustomTotal / float64(totalNodes)
	}
	if totalCPU > 0 {
		summary.CostPerCPU = costBreakdown.CustomTotal / float64(totalCPU)
	}
	if totalMemory > 0 {
		summary.CostPerGB = costBreakdown.CustomTotal / float64(totalMemory/1024/1024/1024) // Convert to GB
	}

	report.TotalCost = *costBreakdown
	report.Summary = *summary

	return report, nil
}

// GenerateOptimizationSuggestions generates cost optimization recommendations
func (c *DefaultCostCalculator) GenerateOptimizationSuggestions(report *CostReport) *CostOptimization {
	optimization := &CostOptimization{
		CurrentCost:   report.TotalCost.CustomTotal,
		Suggestions:   make([]OptimizationSuggestion, 0),
	}

	// Suggestion 1: Check for underutilized nodes
	if len(report.Nodes) > 1 {
		suggestion := OptimizationSuggestion{
			Type:        "rightsizing",
			Description: "일부 노드의 사용률이 낮을 수 있습니다. 인스턴스 타입을 더 작은 것으로 변경을 고려해보세요.",
			Impact:      "medium",
			Saving:      report.TotalCost.CustomTotal * 0.2, // Estimated 20% saving
			Effort:      "medium",
		}
		optimization.Suggestions = append(optimization.Suggestions, suggestion)
	}

	// Suggestion 2: Check for instance type diversity
	instanceTypes := make(map[string]int)
	for _, node := range report.Nodes {
		instanceTypes[node.PricingInfo.InstanceType]++
	}

	if len(instanceTypes) > 3 {
		suggestion := OptimizationSuggestion{
			Type:        "standardization",
			Description: "다양한 인스턴스 타입이 사용되고 있습니다. 표준 인스턴스 타입으로 통일하면 관리가 쉬워집니다.",
			Impact:      "low",
			Saving:      report.TotalCost.CustomTotal * 0.1, // Estimated 10% saving
			Effort:      "high",
		}
		optimization.Suggestions = append(optimization.Suggestions, suggestion)
	}

	// Suggestion 3: Check for spot instances opportunities
	if report.Summary.RunningNodes > 2 {
		suggestion := OptimizationSuggestion{
			Type:        "spot_instances",
			Description: "일부 워크로드를 스팟 인스턴스로 전환하여 비용을 절약할 수 있습니다.",
			Impact:      "high",
			Saving:      report.TotalCost.CustomTotal * 0.4, // Estimated 40% saving
			Effort:      "medium",
		}
		optimization.Suggestions = append(optimization.Suggestions, suggestion)
	}

	// Calculate total potential savings
	var totalSaving float64
	for _, suggestion := range optimization.Suggestions {
		totalSaving += suggestion.Saving
	}

	optimization.PotentialSaving = totalSaving
	optimization.OptimizedCost = optimization.CurrentCost - totalSaving

	return optimization
}

// Helper functions

// parseCPU parses CPU string (e.g., "2", "2000m") to integer
func parseCPU(cpuStr string) int {
	if cpuStr == "" {
		return 0
	}

	// Handle millicpu format (e.g., "2000m")
	if strings.HasSuffix(cpuStr, "m") {
		milliStr := strings.TrimSuffix(cpuStr, "m")
		if milli, err := strconv.Atoi(milliStr); err == nil {
			return milli / 1000
		}
	}

	// Handle regular format (e.g., "2")
	if cpu, err := strconv.Atoi(cpuStr); err == nil {
		return cpu
	}

	return 0
}

// parseMemory parses memory string (e.g., "4Gi", "4096Mi") to bytes
func parseMemory(memoryStr string) int64 {
	if memoryStr == "" {
		return 0
	}

	memoryStr = strings.ToUpper(memoryStr)

	// Handle different memory units
	multipliers := map[string]int64{
		"KI": 1024,
		"MI": 1024 * 1024,
		"GI": 1024 * 1024 * 1024,
		"TI": 1024 * 1024 * 1024 * 1024,
		"K":  1000,
		"M":  1000 * 1000,
		"G":  1000 * 1000 * 1000,
		"T":  1000 * 1000 * 1000 * 1000,
	}

	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(memoryStr, suffix) {
			numStr := strings.TrimSuffix(memoryStr, suffix)
			if num, err := strconv.ParseInt(numStr, 10, 64); err == nil {
				return num * multiplier
			}
		}
	}

	// Try to parse as plain number (assuming bytes)
	if num, err := strconv.ParseInt(memoryStr, 10, 64); err == nil {
		return num
	}

	return 0
}

// calculateKubernetesServiceCosts calculates additional Kubernetes service costs
func (c *DefaultCostCalculator) calculateKubernetesServiceCosts(costBreakdown *CostBreakdown, duration time.Duration, actualUptime time.Duration) error {
	// Get Kubernetes service pricing from repository
	if configRepo, ok := c.pricingRepo.(*ConfigPricingRepository); ok {
		servicePricing, err := configRepo.GetKubernetesServicePricing(csp.CSPTypeNaver)
		if err != nil {
			return fmt.Errorf("failed to get Kubernetes service pricing: %w", err)
		}

		// Calculate cluster management costs
		clusterCost := c.calculateServiceCost(servicePricing.ClusterManagementFee, duration, actualUptime)
		clusterCost.Description = servicePricing.ClusterManagementDescription

		// Calculate load balancer costs (assuming 1 load balancer per cluster)
		lbCost := c.calculateServiceCost(servicePricing.LoadBalancerFee, duration, actualUptime)
		lbCost.Description = servicePricing.LoadBalancerDescription

		// Calculate total service costs
		totalServiceCost := ServiceCost{
			HourlyCost:      clusterCost.HourlyCost + lbCost.HourlyCost,
			DailyCost:       clusterCost.DailyCost + lbCost.DailyCost,
			MonthlyCost:     clusterCost.MonthlyCost + lbCost.MonthlyCost,
			YearlyCost:      clusterCost.YearlyCost + lbCost.YearlyCost,
			CustomCost:      clusterCost.CustomCost + lbCost.CustomCost,
			ActualUsageCost: clusterCost.ActualUsageCost + lbCost.ActualUsageCost,
			Description:     "Total Kubernetes Service Costs",
		}

		// Set Kubernetes service costs
		costBreakdown.KubernetesServiceCost = KubernetesServiceCost{
			ClusterManagementCost: clusterCost,
			LoadBalancerCost:      lbCost,
			TotalServiceCost:      totalServiceCost,
		}

		// Add service costs to total breakdown
		costBreakdown.HourlyTotal += totalServiceCost.HourlyCost
		costBreakdown.DailyTotal += totalServiceCost.DailyCost
		costBreakdown.MonthlyTotal += totalServiceCost.MonthlyCost
		costBreakdown.YearlyTotal += totalServiceCost.YearlyCost
		costBreakdown.CustomTotal += totalServiceCost.CustomCost
		costBreakdown.ActualUsageTotal += totalServiceCost.ActualUsageCost
	}

	return nil
}

// calculateServiceCost calculates cost for a specific service
func (c *DefaultCostCalculator) calculateServiceCost(hourlyPrice float64, duration time.Duration, actualUptime time.Duration) ServiceCost {
	return ServiceCost{
		HourlyCost:      hourlyPrice,
		DailyCost:       hourlyPrice * 24,
		MonthlyCost:     hourlyPrice * 24 * 30,
		YearlyCost:      hourlyPrice * 24 * 365,
		CustomCost:      hourlyPrice * duration.Hours(),
		ActualUsageCost: hourlyPrice * actualUptime.Hours(),
	}
}