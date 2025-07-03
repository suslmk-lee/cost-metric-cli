package calculator

import (
	"fmt"
	"time"

	"github.com/minkyulee/cost-metric-cli/pkg/csp"
	"github.com/spf13/viper"
)

// ConfigPricingRepository implements PricingRepository using configuration files
type ConfigPricingRepository struct {
	config *viper.Viper
}

// NewConfigPricingRepository creates a new configuration-based pricing repository
func NewConfigPricingRepository(configPath string) (*ConfigPricingRepository, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return &ConfigPricingRepository{
		config: v,
	}, nil
}

// GetPricing retrieves pricing information for a specific instance type
func (r *ConfigPricingRepository) GetPricing(cspType csp.CSPType, instanceType, region string) (*PricingInfo, error) {
	cspKey := string(cspType)
	
	// Check if CSP exists in config
	if !r.config.IsSet(cspKey) {
		return nil, fmt.Errorf("CSP %s not found in configuration", cspType)
	}

	// Get instance types for the CSP
	instanceTypesKey := fmt.Sprintf("%s.instance_types", cspKey)
	if !r.config.IsSet(instanceTypesKey) {
		return nil, fmt.Errorf("instance types not found for CSP %s", cspType)
	}

	// Get specific instance type
	instanceKey := fmt.Sprintf("%s.instance_types.%s", cspKey, instanceType)
	if !r.config.IsSet(instanceKey) {
		return nil, fmt.Errorf("instance type %s not found for CSP %s", instanceType, cspType)
	}

	// Extract pricing information
	pricingInfo := &PricingInfo{
		InstanceType: instanceType,
		Region:       region,
		UpdatedAt:    time.Now(),
	}

	// Get currency
	currencyKey := fmt.Sprintf("%s.default_currency", cspKey)
	pricingInfo.Currency = r.config.GetString(currencyKey)
	if pricingInfo.Currency == "" {
		pricingInfo.Currency = "KRW"
	}

	// Get pricing details
	pricingInfo.HourlyPrice = r.config.GetFloat64(fmt.Sprintf("%s.hourly_price", instanceKey))
	pricingInfo.CPU = r.config.GetInt(fmt.Sprintf("%s.cpu", instanceKey))
	pricingInfo.Memory = r.config.GetInt(fmt.Sprintf("%s.memory", instanceKey))

	if pricingInfo.HourlyPrice == 0 {
		return nil, fmt.Errorf("pricing information not available for instance type %s", instanceType)
	}

	return pricingInfo, nil
}

// GetAllPricing retrieves all pricing information for a CSP and region
func (r *ConfigPricingRepository) GetAllPricing(cspType csp.CSPType, region string) (map[string]*PricingInfo, error) {
	cspKey := string(cspType)
	
	if !r.config.IsSet(cspKey) {
		return nil, fmt.Errorf("CSP %s not found in configuration", cspType)
	}

	instanceTypesKey := fmt.Sprintf("%s.instance_types", cspKey)
	instanceTypes := r.config.GetStringMap(instanceTypesKey)
	
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
	cspKey := string(cspType)
	
	if !r.config.IsSet(cspKey) {
		return nil, fmt.Errorf("CSP %s not found in configuration", cspType)
	}

	instanceTypesKey := fmt.Sprintf("%s.instance_types", cspKey)
	instanceTypes := r.config.GetStringMap(instanceTypesKey)
	
	var types []string
	for instanceType := range instanceTypes {
		types = append(types, instanceType)
	}

	return types, nil
}

// GetCSPInfo returns basic information about a CSP
func (r *ConfigPricingRepository) GetCSPInfo(cspType csp.CSPType) (*CSPConfig, error) {
	cspKey := string(cspType)
	
	if !r.config.IsSet(cspKey) {
		return nil, fmt.Errorf("CSP %s not found in configuration", cspType)
	}

	config := &CSPConfig{
		Type:     cspType,
		Name:     r.config.GetString(fmt.Sprintf("%s.name", cspKey)),
		APIUrl:   r.config.GetString(fmt.Sprintf("%s.api_url", cspKey)),
		Currency: r.config.GetString(fmt.Sprintf("%s.default_currency", cspKey)),
	}

	// Get regions
	regionsKey := fmt.Sprintf("%s.regions", cspKey)
	if r.config.IsSet(regionsKey) {
		var regions []Region
		regionsList := r.config.Get(regionsKey)
		
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
	cspKey := string(cspType)
	
	if !r.config.IsSet(cspKey) {
		return nil, fmt.Errorf("CSP %s not found in configuration", cspType)
	}

	serviceKey := fmt.Sprintf("%s.kubernetes_service", cspKey)
	if !r.config.IsSet(serviceKey) {
		return nil, fmt.Errorf("Kubernetes service pricing not found for CSP %s", cspType)
	}

	pricing := &KubernetesServicePricing{
		Currency: r.config.GetString(fmt.Sprintf("%s.default_currency", cspKey)),
	}

	// Get cluster management fee
	clusterKey := fmt.Sprintf("%s.cluster_management_fee", serviceKey)
	if r.config.IsSet(clusterKey) {
		pricing.ClusterManagementFee = r.config.GetFloat64(fmt.Sprintf("%s.hourly_price", clusterKey))
		pricing.ClusterManagementDescription = r.config.GetString(fmt.Sprintf("%s.description", clusterKey))
	}

	// Get load balancer cost
	lbKey := fmt.Sprintf("%s.load_balancer", serviceKey)
	if r.config.IsSet(lbKey) {
		pricing.LoadBalancerFee = r.config.GetFloat64(fmt.Sprintf("%s.hourly_price", lbKey))
		pricing.LoadBalancerDescription = r.config.GetString(fmt.Sprintf("%s.description", lbKey))
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