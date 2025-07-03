package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minkyulee/cost-metric-cli/pkg/calculator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze current Kubernetes context costs",
	Long:  `Analyze the costs of the current Kubernetes context by detecting the CSP and calculating instance costs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCostAnalysis(cmd)
	},
}

var listContextsCmd = &cobra.Command{
	Use:   "list-contexts",
	Short: "List available Kubernetes contexts",
	Long:  `Display all available Kubernetes contexts from kubeconfig.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListContexts()
	},
}

var pricingCmd = &cobra.Command{
	Use:   "pricing",
	Short: "Show pricing information",
	Long:  `Display pricing information for different CSP instance types.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShowPricing(cmd)
	},
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate detailed cost report",
	Long:  `Generate a detailed cost report for the current or specified Kubernetes context.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGenerateReport(cmd)
	},
}

// runCostAnalysis performs cost analysis
func runCostAnalysis(cmd *cobra.Command) error {
	fmt.Println("🔍 비용 분석을 시작합니다...")

	// Get configuration
	analyzer, err := getAnalyzer()
	if err != nil {
		return fmt.Errorf("analyzer 초기화 실패: %w", err)
	}

	// Get options from flags
	contextName := viper.GetString("context")
	durationStr := viper.GetString("duration")

	var region string
	var verbose bool

	// Safely get flags if cmd is not nil
	if cmd != nil {
		region, _ = cmd.Flags().GetString("region")
		verbose, _ = cmd.Flags().GetBool("verbose")
	}

	// Parse duration
	duration, err := parseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("duration 파싱 실패: %w", err)
	}

	// Get current context if not specified
	if contextName == "" {
		contextName, err = analyzer.GetCurrentContext()
		if err != nil {
			return fmt.Errorf("현재 컨텍스트 조회 실패: %w", err)
		}
	}

	fmt.Printf("📊 컨텍스트: %s\n", contextName)
	fmt.Printf("⏱️  분석 기간: %s\n", duration)

	// Perform analysis
	ctx := context.Background()
	options := calculator.AnalysisOptions{
		Context:  contextName,
		Duration: duration,
		Region:   region,
		Verbose:  verbose,
	}

	report, err := analyzer.QuickAnalysis(ctx, options)
	if err != nil {
		return fmt.Errorf("비용 분석 실패: %w", err)
	}

	// Display results
	displayCostReport(report, verbose)

	return nil
}

// runListContexts lists available Kubernetes contexts
func runListContexts() error {
	fmt.Println("🔍 사용 가능한 Kubernetes 컨텍스트:")

	analyzer, err := getAnalyzer()
	if err != nil {
		return fmt.Errorf("analyzer 초기화 실패: %w", err)
	}

	contexts, err := analyzer.ListAvailableContexts()
	if err != nil {
		return fmt.Errorf("컨텍스트 목록 조회 실패: %w", err)
	}

	currentContext, _ := analyzer.GetCurrentContext()

	for i, context := range contexts {
		marker := " "
		if context == currentContext {
			marker = "*"
		}
		fmt.Printf("%s %d. %s\n", marker, i+1, context)
	}

	if currentContext != "" {
		fmt.Printf("\n현재 컨텍스트: %s\n", currentContext)
	}

	return nil
}

// runShowPricing displays pricing information
func runShowPricing(cmd *cobra.Command) error {
	fmt.Println("💰 가격 정보를 조회합니다...")

	analyzer, err := getAnalyzer()
	if err != nil {
		return fmt.Errorf("analyzer 초기화 실패: %w", err)
	}

	// Get supported CSPs
	csps, err := analyzer.GetSupportedCSPs()
	if err != nil {
		return fmt.Errorf("지원 CSP 조회 실패: %w", err)
	}

	for _, cspConfig := range csps {
		fmt.Printf("\n🏢 %s (%s)\n", cspConfig.Name, cspConfig.Type)
		fmt.Printf("   통화: %s\n", cspConfig.Currency)

		// Get pricing for each region
		for _, region := range cspConfig.Regions {
			fmt.Printf("\n   📍 지역: %s (%s)\n", region.DisplayName, region.Name)

			pricingMap, err := analyzer.GetPricingInfo(cspConfig.Type, region.Name)
			if err != nil {
				fmt.Printf("      ❌ 가격 정보 조회 실패: %v\n", err)
				continue
			}

			fmt.Printf("      %-20s %-15s %-10s %s\n", "인스턴스 타입", "시간당 가격", "CPU", "메모리")
			fmt.Printf("      %s\n", strings.Repeat("-", 60))

			for instanceType, pricing := range pricingMap {
				fmt.Printf("      %-20s %-15.0f %-10d %dMB\n",
					instanceType,
					pricing.HourlyPrice,
					pricing.CPU,
					pricing.Memory)
			}
		}
	}

	return nil
}

// runGenerateReport generates a detailed cost report
func runGenerateReport(cmd *cobra.Command) error {
	fmt.Println("📋 상세 비용 리포트를 생성합니다...")

	// Run analysis first
	if err := runCostAnalysis(cmd); err != nil {
		return err
	}

	// Get output path
	outputPath := viper.GetString("output")
	if outputPath == "" {
		outputPath = fmt.Sprintf("cost-report-%s.json", time.Now().Format("20060102-150405"))
	}

	// Get analyzer and perform analysis
	analyzer, err := getAnalyzer()
	if err != nil {
		return fmt.Errorf("analyzer 초기화 실패: %w", err)
	}

	contextName := viper.GetString("context")
	if contextName == "" {
		contextName, err = analyzer.GetCurrentContext()
		if err != nil {
			return fmt.Errorf("현재 컨텍스트 조회 실패: %w", err)
		}
	}

	durationStr := viper.GetString("duration")
	duration, err := parseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("duration 파싱 실패: %w", err)
	}

	ctx := context.Background()
	options := calculator.AnalysisOptions{
		Context:  contextName,
		Duration: duration,
		Verbose:  true,
	}

	report, err := analyzer.QuickAnalysis(ctx, options)
	if err != nil {
		return fmt.Errorf("비용 분석 실패: %w", err)
	}

	// Generate optimization suggestions
	optimization := analyzer.GenerateOptimizationReport(report)

	// Create full report with optimization
	fullReport := struct {
		*calculator.CostReport
		Optimization *calculator.CostOptimization `json:"optimization"`
	}{
		CostReport:   report,
		Optimization: optimization,
	}

	// Save to file
	format := viper.GetString("format")
	if err := saveReport(fullReport, outputPath, format); err != nil {
		return fmt.Errorf("리포트 저장 실패: %w", err)
	}

	fmt.Printf("✅ 리포트가 저장되었습니다: %s\n", outputPath)

	return nil
}

// Helper functions

// getAnalyzer creates and returns a cost analyzer
func getAnalyzer() (*calculator.CostAnalyzer, error) {
	// Try multiple possible config directory paths
	possiblePaths := []string{
		filepath.Join("configs", "pricing"),
		filepath.Join("..", "configs", "pricing"),
		filepath.Join(".", "configs", "pricing"),
	}

	// Find the executable directory and try relative to it
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths,
			filepath.Join(execDir, "configs", "pricing"),
			filepath.Join(execDir, "..", "configs", "pricing"),
			filepath.Join(filepath.Dir(execDir), "configs", "pricing"),
		)
	}

	for _, configDir := range possiblePaths {
		if stat, err := os.Stat(configDir); err == nil && stat.IsDir() {
			return calculator.NewCostAnalyzer(configDir)
		}
	}

	return nil, fmt.Errorf("pricing 설정 디렉토리를 찾을 수 없습니다. 다음 경로에서 'configs/pricing' 디렉토리를 확인하세요: %v", possiblePaths)
}

// parseDuration parses duration string (1h, 1d, 1m, 1y)
func parseDuration(durationStr string) (time.Duration, error) {
	if durationStr == "" {
		return time.Hour, nil
	}

	// Handle special cases
	switch strings.ToLower(durationStr) {
	case "1d", "24h":
		return 24 * time.Hour, nil
	case "1w", "7d":
		return 7 * 24 * time.Hour, nil
	case "1m", "30d":
		return 30 * 24 * time.Hour, nil
	case "1y", "365d":
		return 365 * 24 * time.Hour, nil
	}

	// Try to parse as standard duration
	return time.ParseDuration(durationStr)
}

// displayCostReport displays the cost report in a formatted way
func displayCostReport(report *calculator.CostReport, verbose bool) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📊 비용 분석 결과\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	fmt.Printf("🏢 CSP: %s\n", report.CSP.Name)
	fmt.Printf("🎯 클러스터: %s\n", report.ClusterName)
	fmt.Printf("📅 분석 시간: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("⏱️  분석 기간: %s\n", report.AnalysisPeriod)

	fmt.Printf("\n💰 총 비용 (%s)\n", report.TotalCost.Currency)
	fmt.Printf("   시간당: %.0f원\n", report.TotalCost.HourlyTotal)
	fmt.Printf("   일일:   %.0f원\n", report.TotalCost.DailyTotal)
	fmt.Printf("   월간:   %.0f원\n", report.TotalCost.MonthlyTotal)
	fmt.Printf("   연간:   %.0f원\n", report.TotalCost.YearlyTotal)
	fmt.Printf("   분석기간: %.0f원\n", report.TotalCost.CustomTotal)

	fmt.Printf("\n💸 실제 사용 비용 (노드 생성 시점부터)\n")
	fmt.Printf("   총 가동 시간: %.1f시간\n", report.TotalCost.TotalUptimeHours)
	fmt.Printf("   노드 실제 사용 비용: %.0f원\n", report.TotalCost.ActualUsageTotal-report.TotalCost.KubernetesServiceCost.TotalServiceCost.ActualUsageCost)

	fmt.Printf("\n🎛️  Kubernetes 서비스 비용\n")
	fmt.Printf("   클러스터 관리 비용: %.0f원 (시간당 %.0f원)\n",
		report.TotalCost.KubernetesServiceCost.ClusterManagementCost.ActualUsageCost,
		report.TotalCost.KubernetesServiceCost.ClusterManagementCost.HourlyCost)
	fmt.Printf("   로드밸런서 비용: %.0f원 (시간당 %.0f원)\n",
		report.TotalCost.KubernetesServiceCost.LoadBalancerCost.ActualUsageCost,
		report.TotalCost.KubernetesServiceCost.LoadBalancerCost.HourlyCost)
	fmt.Printf("   서비스 총 비용: %.0f원\n", report.TotalCost.KubernetesServiceCost.TotalServiceCost.ActualUsageCost)

	fmt.Printf("\n💰 전체 실제 사용 비용: %.0f원\n", report.TotalCost.ActualUsageTotal)

	fmt.Printf("\n📈 클러스터 요약\n")
	fmt.Printf("   총 노드: %d개 (실행중: %d개)\n", report.Summary.TotalNodes, report.Summary.RunningNodes)
	fmt.Printf("   총 CPU: %d코어\n", report.Summary.TotalCPU)
	fmt.Printf("   총 메모리: %.1fGB\n", float64(report.Summary.TotalMemory)/1024/1024/1024)
	fmt.Printf("   노드당 평균 비용: %.0f원\n", report.Summary.AvgCostPerNode)

	if verbose {
		fmt.Printf("\n📋 노드별 상세 정보\n")
		// 헤더 컬럼 정렬 및 너비 조정
		fmt.Printf("%-21s %-16s %-11s %-13s %13s %16s %20s\n", "노드명", "인스턴스타입", "상태", "가동시간", "시간당비용", "분석기간비용", "실제사용비용")
		fmt.Printf("%s\n", strings.Repeat("-", 110))

		for _, node := range report.Nodes {
			uptimeStr := formatDuration(node.ActualUptime)
			actualUsageCostStr := fmt.Sprintf("%.0f원", node.ActualUsageCost)
			// 데이터 컬럼 정렬 및 너비 조정
			fmt.Printf("%-21s %-16s %-11s %-13s %13.0f %16.0f %20s\n",
				truncateString(node.NodeInfo.Name, 20),
				truncateString(node.NodeInfo.InstanceType, 15),
				node.NodeInfo.Status,
				uptimeStr,
				node.HourlyCost,
				node.CustomCost,
				actualUsageCostStr)
		}

		fmt.Printf("\n📊 인스턴스 타입별 비용\n")
		for instanceType, cost := range report.TotalCost.ByType {
			fmt.Printf("   %s: %.0f원\n", instanceType, cost)
		}
	}
}

// saveReport saves the report to a file
func saveReport(report interface{}, outputPath, format string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(report, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(report)
	default:
		data, err = json.MarshalIndent(report, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("데이터 마샬링 실패: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// truncateString truncates a string to specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// runCostAnalysisInteractive performs cost analysis for interactive mode
func runCostAnalysisInteractive() error {
	fmt.Println("🔍 비용 분석을 시작합니다...")

	// Get configuration
	analyzer, err := getAnalyzer()
	if err != nil {
		return fmt.Errorf("analyzer 초기화 실패: %w", err)
	}

	// Get options with defaults for interactive mode
	contextName := viper.GetString("context")
	durationStr := viper.GetString("duration")
	if durationStr == "" {
		durationStr = "1h" // Default duration for interactive mode
	}

	// Parse duration
	duration, err := parseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("duration 파싱 실패: %w", err)
	}

	// Get current context if not specified
	if contextName == "" {
		contextName, err = analyzer.GetCurrentContext()
		if err != nil {
			return fmt.Errorf("현재 컨텍스트 조회 실패: %w", err)
		}
	}

	fmt.Printf("📊 컨텍스트: %s\n", contextName)
	fmt.Printf("⏱️  분석 기간: %s\n", duration)

	// Perform analysis
	ctx := context.Background()
	options := calculator.AnalysisOptions{
		Context:  contextName,
		Duration: duration,
		Verbose:  true, // Always verbose in interactive mode
	}

	report, err := analyzer.QuickAnalysis(ctx, options)
	if err != nil {
		return fmt.Errorf("비용 분석 실패: %w", err)
	}

	// Display results
	displayCostReport(report, true)

	return nil
}

// runShowPricingInteractive displays pricing information for interactive mode
func runShowPricingInteractive() error {
	fmt.Println("💰 가격 정보를 조회합니다...")

	analyzer, err := getAnalyzer()
	if err != nil {
		return fmt.Errorf("analyzer 초기화 실패: %w", err)
	}

	// Get supported CSPs
	csps, err := analyzer.GetSupportedCSPs()
	if err != nil {
		return fmt.Errorf("지원 CSP 조회 실패: %w", err)
	}

	for _, cspConfig := range csps {
		fmt.Printf("\n🏢 %s (%s)\n", cspConfig.Name, cspConfig.Type)
		fmt.Printf("   통화: %s\n", cspConfig.Currency)

		// Get pricing for each region
		for _, region := range cspConfig.Regions {
			fmt.Printf("\n   📍 지역: %s (%s)\n", region.DisplayName, region.Name)

			pricingMap, err := analyzer.GetPricingInfo(cspConfig.Type, region.Name)
			if err != nil {
				fmt.Printf("      ❌ 가격 정보 조회 실패: %v\n", err)
				continue
			}

			fmt.Printf("      %-20s %-15s %-10s %s\n", "인스턴스 타입", "시간당 가격", "CPU", "메모리")
			fmt.Printf("      %s\n", strings.Repeat("-", 60))

			for instanceType, pricing := range pricingMap {
				fmt.Printf("      %-20s %-15.0f %-10d %dMB\n",
					instanceType,
					pricing.HourlyPrice,
					pricing.CPU,
					pricing.Memory)
			}
		}
	}

	return nil
}

// runListContextsInteractive lists available Kubernetes contexts for interactive mode
func runListContextsInteractive() error {
	fmt.Println("🔍 사용 가능한 Kubernetes 컨텍스트:")

	analyzer, err := getAnalyzer()
	if err != nil {
		return fmt.Errorf("analyzer 초기화 실패: %w", err)
	}

	contexts, err := analyzer.ListAvailableContexts()
	if err != nil {
		return fmt.Errorf("컨텍스트 목록 조회 실패: %w", err)
	}

	currentContext, _ := analyzer.GetCurrentContext()

	for i, context := range contexts {
		marker := " "
		if context == currentContext {
			marker = "*"
		}
		fmt.Printf("%s %d. %s\n", marker, i+1, context)
	}

	if currentContext != "" {
		fmt.Printf("\n현재 컨텍스트: %s\n", currentContext)
	}

	return nil
}

// formatDuration formats duration into human readable string
func formatDuration(d time.Duration) string {
	hours := d.Hours()
	if hours < 24 {
		return fmt.Sprintf("%.1fh", hours)
	}
	days := hours / 24
	if days < 30 {
		return fmt.Sprintf("%.1fd", days)
	}
	months := days / 30
	return fmt.Sprintf("%.1fm", months)
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(listContextsCmd)
	rootCmd.AddCommand(pricingCmd)
	rootCmd.AddCommand(reportCmd)

	// Add flags specific to analyze command
	analyzeCmd.Flags().StringP("region", "r", "", "Specify region for cost calculation")
	analyzeCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	// Add flags specific to report command
	reportCmd.Flags().StringP("template", "t", "default", "Report template (default, detailed)")
	reportCmd.Flags().BoolP("export", "e", false, "Export report to file")
}
