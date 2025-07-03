package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minkyulee/cost-metric-cli/pkg/csp"
	"github.com/spf13/viper"
)

// ConfigPricingRepository implements PricingRepository using configuration files
type ConfigPricingRepository struct {
	configs map[csp.CSPType]*viper.Viper
}

// NewConfigPricingRepository creates a new configuration-based pricing repository
func NewConfigPricingRepository(configDir string) (*ConfigPricingRepository, error) {
	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory %s: %w", configDir, err)
	}

	configs := make(map[csp.CSPType]*viper.Viper)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := filepath.Ext(file.Name())
		if ext == ".yaml" || ext == ".yml" {
			cspTypeStr := strings.TrimSuffix(file.Name(), ext)
			cspType := csp.CSPType(cspTypeStr)

			v := viper.New()
			v.SetConfigFile(filepath.Join(configDir, file.Name()))
			v.SetConfigType("yaml")

			if err := v.ReadInConfig(); err != nil {
				fmt.Printf("Warning: failed to read pricing config file %s: %v\n", file.Name(), err)
				continue
			}
			configs[cspType] = v
		}
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no valid pricing config files found in %s", configDir)
	}
	return &ConfigPricingRepository{
		configs: configs,
	}, nil
}

// GetPricing retrieves pricing information for a specific instance type
func (r *ConfigPricingRepository) GetPricing(cspType csp.CSPType, instanceType, region string) (*PricingInfo, error) {
	v, ok := r.configs[cspType]
	if !ok {
		return nil, fmt.Errorf("pricing configuration for CSP %s not found", cspType)
	}

	// Get instance types for the CSP
	instanceTypesKey := "instance_types"
	if !v.IsSet(instanceTypesKey) {
		return nil, fmt.Errorf("instance types not found for CSP %s", cspType)
	}

	// Get specific instance type
	instanceKey := fmt.Sprintf("%s.%s", instanceTypesKey, instanceType)
	if !v.IsSet(instanceKey) {
		return nil, fmt.Errorf("instance type %s not found for CSP %s", instanceType, cspType)
	}

	// Extract pricing information
	pricingInfo := &PricingInfo{
		InstanceType: instanceType,
		Region:       region,
		UpdatedAt:    time.Now(),
	}

	// Get currency
	pricingInfo.Currency = v.GetString("default_currency")
	if pricingInfo.Currency == "" {
		pricingInfo.Currency = "KRW"
	}

	// Get pricing details
	pricingInfo.HourlyPrice = v.GetFloat64(fmt.Sprintf("%s.hourly_price", instanceKey))
	pricingInfo.CPU = v.GetInt(fmt.Sprintf("%s.cpu", instanceKey))
	pricingInfo.Memory = v.GetInt(fmt.Sprintf("%s.memory", instanceKey))

	if pricingInfo.HourlyPrice == 0 {
		return nil, fmt.Errorf("pricing information not available for instance type %s", instanceType)
	}

	return pricingInfo, nil
}

// GetAllPricing retrieves all pricing information for a CSP and region
func (r *ConfigPricingRepository) GetAllPricing(cspType csp.CSPType, region string) (map[string]*PricingInfo, error) {
	v, ok := r.configs[cspType]
	if !ok {
		return nil, fmt.Errorf("pricing configuration for CSP %s not found", cspType)
	}

	instanceTypesKey := "instance_types"
	instanceTypes := v.GetStringMap(instanceTypesKey)

	if len(instanceTypes) == 0 {
		return nil, fmt.Errorf("no instance types found for CSP %s", cspType)
	}

	pricingMap := make(map[string]*PricingInfo)

	for instanceType := range instanceTypes {
		pricing, err := r.GetPricing(cspType, instanceType, region)
		if err != nil {
			// Log error but continue with other instance types
			continue
		}
		pricingMap[instanceType] = pricing
	}

	return pricingMap, nil
}

// UpdatePricing updates pricing information (placeholder for API implementation)
func (r *ConfigPricingRepository) UpdatePricing(cspType csp.CSPType, region string) error {
	// TODO: Implement API-based pricing updates
	// For now, this is a no-op as we're using static configuration
	return nil
}

// GetSupportedInstanceTypes returns all supported instance types for a CSP
func (r *ConfigPricingRepository) GetSupportedInstanceTypes(cspType csp.CSPType) ([]string, error) {
	v, ok := r.configs[cspType]
	if !ok {
		return nil, fmt.Errorf("pricing configuration for CSP %s not found", cspType)
	}

	instanceTypesKey := "instance_types"
	instanceTypes := v.GetStringMap(instanceTypesKey)

	var types []string
	for instanceType := range instanceTypes {
		types = append(types, instanceType)
	}

	return types, nil
}

// GetCSPInfo returns basic information about a CSP
func (r *ConfigPricingRepository) GetCSPInfo(cspType csp.CSPType) (*CSPConfig, error) {
	v, ok := r.configs[cspType]
	if !ok {
		return nil, fmt.Errorf("pricing configuration for CSP %s not found", cspType)
	}

	config := &CSPConfig{
		Type:     cspType,
		Name:     v.GetString("name"),
		APIUrl:   v.GetString("api_url"),
		Currency: v.GetString("default_currency"),
	}

	// Get regions
	regionsKey := "regions"
	if v.IsSet(regionsKey) {
		var regions []Region
		regionsList := v.Get(regionsKey)

		if regionSlice, ok := regionsList.([]interface{}); ok {
			for _, regionItem := range regionSlice {
				if regionMap, ok := regionItem.(map[string]interface{}); ok {
					region := Region{
						Name:        getStringFromMap(regionMap, "name"),
						DisplayName: getStringFromMap(regionMap, "display_name"),
					}
					regions = append(regions, region)
				}
			}
		}
		config.Regions = regions
	}

	return config, nil
}

// GetKubernetesServicePricing retrieves Kubernetes service pricing information
func (r *ConfigPricingRepository) GetKubernetesServicePricing(cspType csp.CSPType) (*KubernetesServicePricing, error) {
	v, ok := r.configs[cspType]
	if !ok {
		return nil, fmt.Errorf("pricing configuration for CSP %s not found", cspType)
	}

	serviceKey := "kubernetes_service"
	if !v.IsSet(serviceKey) {
		// Not an error, some CSPs may not have this pricing.
		return nil, nil
	}

	pricing := &KubernetesServicePricing{
		Currency: v.GetString("default_currency"),
	}

	// Get cluster management fee
	clusterKey := fmt.Sprintf("%s.cluster_management_fee", serviceKey)
	if v.IsSet(clusterKey) {
		pricing.ClusterManagementFee = v.GetFloat64(fmt.Sprintf("%s.hourly_price", clusterKey))
		pricing.ClusterManagementDescription = v.GetString(fmt.Sprintf("%s.description", clusterKey))
	}

	// Get load balancer cost
	lbKey := fmt.Sprintf("%s.load_balancer", serviceKey)
	if v.IsSet(lbKey) {
		pricing.LoadBalancerFee = v.GetFloat64(fmt.Sprintf("%s.hourly_price", lbKey))
		pricing.LoadBalancerDescription = v.GetString(fmt.Sprintf("%s.description", lbKey))
	}
	return pricing, nil
}

// KubernetesServicePricing represents Kubernetes service pricing
type KubernetesServicePricing struct {
	ClusterManagementFee         float64 `json:"cluster_management_fee"`
	ClusterManagementDescription string  `json:"cluster_management_description"`
	LoadBalancerFee              float64 `json:"load_balancer_fee"`
	LoadBalancerDescription      string  `json:"load_balancer_description"`
	Currency                     string  `json:"currency"`
}

// CSPConfig represents CSP configuration
type CSPConfig struct {
	Type     csp.CSPType `json:"type"`
	Name     string      `json:"name"`
	APIUrl   string      `json:"api_url"`
	Currency string      `json:"currency"`
	Regions  []Region    `json:"regions"`
}

// Region represents a CSP region
type Region struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// Helper function to safely get string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
