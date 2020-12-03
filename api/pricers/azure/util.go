package azure

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

func PricePlanFile(ctx context.Context, jsonBlob string, priceType PricingScheme) (float64, []string, []string, error) {
	var unsupportedResources []string
	var unestimateableResources []string
	var pf common.PlanFile
	err := json.Unmarshal([]byte(jsonBlob), &pf)
	if err != nil {
		klog.Error(err)
		return 0.0, []string{}, []string{}, err
	}
	var hourlyPrice float64
	var resources []common.Priceable

	for _, change := range pf.ResourceChanges {
		//we only want to price Azure API changes
		if change.Provider == "registry.terraform.io/hashicorp/azurerm" {
			//Until I find a better way we need to explicitly opt-in price types
			switch change.Type {
			case "azurerm_linux_virtual_machine":
				resources = append(resources, &VirtualMachine{
					Size:          change.Change.After.(map[string]interface{})["size"].(string),
					Location:      change.Change.After.(map[string]interface{})["location"].(string),
					Count:         1.0,
					IsSpotEnabled: change.Change.After.(map[string]interface{})["priority"].(string) == "Spot",
					IsWindows:     false,
					PricingScheme: priceType,
				})
			case "azurerm_windows_virtual_machine":
				resources = append(resources, &VirtualMachine{
					Size:          change.Change.After.(map[string]interface{})["size"].(string),
					Location:      change.Change.After.(map[string]interface{})["location"].(string),
					Count:         1.0,
					IsSpotEnabled: change.Change.After.(map[string]interface{})["priority"].(string) == "Spot",
					IsWindows:     true,
					PricingScheme: priceType,
				})
			case "azurerm_kubernetes_cluster":
				resources = append(resources, &AksCluster{
					IsPaid: change.Change.After.(map[string]interface{})["sku_tier"].(string) == "Paid",
				},
					&VirtualMachine{
						Size:      change.Change.After.(map[string]interface{})["default_node_pool"].([]interface{})[0].(map[string]interface{})["vm_size"].(string),
						Location:  change.Change.After.(map[string]interface{})["location"].(string),
						Count:     change.Change.After.(map[string]interface{})["default_node_pool"].([]interface{})[0].(map[string]interface{})["node_count"].(float64),
						IsWindows: false,
					})
			//This is where a resource that is unsupported	will fall through
			case "azurerm_subnet":
				unestimateableResources = append(unestimateableResources, "azurerm_subnet")
				break
			case "azurerm_resource_group":
				unestimateableResources = append(unestimateableResources, "azurerm_resource_group")
				break
			case "azurerm_virtual_network":
				unestimateableResources = append(unestimateableResources, "azurerm_virtual_network")
				break
			case "azurerm_network_interface":
				unestimateableResources = append(unestimateableResources, "azurerm_network_interface")
				break
			case "azurerm_virtual_machine_scale_set":
				var isWindows bool
				if len(change.Change.After.(map[string]interface{})["os_profile_windows_config"].([]interface{})) > 0 {
					isWindows = true
				}
				resources = append(resources, &VirtualMachine{
					IsWindows: isWindows,
					Count:     change.Change.After.(map[string]interface{})["sku"].([]interface{})[0].(map[string]interface{})["capacity"].(float64),
					Size:      change.Change.After.(map[string]interface{})["sku"].([]interface{})[0].(map[string]interface{})["name"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
				})
				break
			case "azurerm_virtual_machine":
				var isWindows bool
				if len(change.Change.After.(map[string]interface{})["os_profile_windows_config"].([]interface{})) > 0 {
					isWindows = true
				}
				resources = append(resources, &VirtualMachine{
					IsWindows: isWindows,
					Count:     1,
					Size:      change.Change.After.(map[string]interface{})["vm_size"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
				})
				break
			case "azurerm_windows_virtual_machine_scale_set":
				resources = append(resources, &VirtualMachine{
					IsWindows: true,
					Count:     change.Change.After.(map[string]interface{})["instances"].(float64),
					Size:      change.Change.After.(map[string]interface{})["sku"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
				})
				break
			case "azurerm_linux_virtual_machine_scale_set":
				resources = append(resources, &VirtualMachine{
					IsWindows: false,
					Count:     change.Change.After.(map[string]interface{})["instances"].(float64),
					Size:      change.Change.After.(map[string]interface{})["sku"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
					//TODO make helpers to grab a key of desired type from chage struct
				})
				break
			case "azurerm_managed_disk":
				resources = append(resources, &AzureDisk{
					Location: change.Change.After.(map[string]interface{})["location"].(string),
					SizeInGb: change.Change.After.(map[string]interface{})["disk_size_gb"].(float64),
					SkuTier:  change.Change.After.(map[string]interface{})["storage_account_type"].(string),
					Count:    1,
				})
				break
			default:
				unsupportedResources = append(unsupportedResources, change.Address)
				break
			}
		}
	}

	for _, res := range resources {
		hourlyPrice += res.GetHourlyPrice(ctx)
	}

	return hourlyPrice, unsupportedResources, unestimateableResources, nil
}
