package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/zparnold/azure-terraform-cost-estimator/api/errors"
	"github.com/zparnold/azure-terraform-cost-estimator/api/pricers/azure"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog"
)

const (
	YEAR_HOURS  = 8760
	MONTH_HOURS = 730
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (apiResp Response, err error) {
	//Ensure that we capture and properly handle any panic()'s in the API
	defer func() {
		if r := recover(); r != nil {
			apiResp = generateErrorResp(ctx, 500, "Internal Server Error", fmt.Sprintf("%v", err))
			err = nil
		}
	}()
	//request.RequestContext.Identity.SourceIP
	var r common.ApiResp
	price, unsupportedResources, unestimateableResources, err := PricePlanFile(ctx, request.Body)
	if err != nil {
		apiResp = generateErrorResp(ctx, 500, "Internal Server Error", fmt.Sprintf("%v", err))
		err = nil
		return apiResp, nil
	}
	r.TotalEstimate.HourlyCost = price
	r.TotalEstimate.MonthlyCost = price * MONTH_HOURS
	r.TotalEstimate.YearlyCost = price * YEAR_HOURS
	r.UnsupportedResources = unsupportedResources
	r.UnestimateableResources = unestimateableResources
	b, err := json.Marshal(r)
	if err != nil {
		apiResp = generateErrorResp(ctx, 500, "Internal Server Error", fmt.Sprintf("%v", err))
		err = nil
		return apiResp, nil
	}
	apiResp = Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(b),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return apiResp, nil
}
func PricePlanFile(ctx context.Context, jsonBlob string) (float64, []string, []string, error) {
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
				resources = append(resources, &azure.VirtualMachine{
					Size:          change.Change.After.(map[string]interface{})["size"].(string),
					Location:      change.Change.After.(map[string]interface{})["location"].(string),
					Count:         1.0,
					IsSpotEnabled: change.Change.After.(map[string]interface{})["priority"].(string) == "Spot",
					IsWindows:     false,
				})
			case "azurerm_windows_virtual_machine":
				resources = append(resources, &azure.VirtualMachine{
					Size:          change.Change.After.(map[string]interface{})["size"].(string),
					Location:      change.Change.After.(map[string]interface{})["location"].(string),
					Count:         1.0,
					IsSpotEnabled: change.Change.After.(map[string]interface{})["priority"].(string) == "Spot",
					IsWindows:     true,
				})
			case "azurerm_kubernetes_cluster":
				resources = append(resources, &azure.AksCluster{
					IsPaid: change.Change.After.(map[string]interface{})["sku_tier"].(string) == "Paid",
				},
					&azure.VirtualMachine{
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
				resources = append(resources, &azure.VirtualMachine{
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
				resources = append(resources, &azure.VirtualMachine{
					IsWindows: isWindows,
					Count:     1,
					Size:      change.Change.After.(map[string]interface{})["vm_size"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
				})
				break
			case "azurerm_windows_virtual_machine_scale_set":
				resources = append(resources, &azure.VirtualMachine{
					IsWindows: true,
					Count:     change.Change.After.(map[string]interface{})["instances"].(float64),
					Size:      change.Change.After.(map[string]interface{})["sku"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
				})
				break
			case "azurerm_linux_virtual_machine_scale_set":
				resources = append(resources, &azure.VirtualMachine{
					IsWindows: false,
					Count:     change.Change.After.(map[string]interface{})["instances"].(float64),
					Size:      change.Change.After.(map[string]interface{})["sku"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
					//TODO make helpers to grab a key of desired type from chage struct
				})
				break
			case "azurerm_managed_disk":
				resources = append(resources, &azure.AzureDisk{
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

func main() {
	lambda.Start(Handler)
//			_ = xray.Configure(xray.Config{ContextMissingStrategy: ctxmissing.NewDefaultLogErrorStrategy()})
//			something := `
//{"format_version":"0.1","terraform_version":"0.13.5","planned_values":{"root_module":{"resources":[{"address":"azurerm_linux_virtual_machine_scale_set.example","mode":"managed","type":"azurerm_linux_virtual_machine_scale_set","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"additional_capabilities":[],"admin_password":"P@ssw0rd12345","admin_ssh_key":[],"admin_username":"adminuser","automatic_os_upgrade_policy":[],"boot_diagnostics":[],"custom_data":null,"data_disk":[],"disable_password_authentication":true,"do_not_run_extensions_on_overprovisioned_machines":false,"encryption_at_host_enabled":null,"eviction_policy":null,"health_probe_id":null,"identity":[],"instances":1,"location":"westeurope","max_bid_price":-1,"name":"example-vmss","network_interface":[{"dns_servers":null,"enable_accelerated_networking":false,"enable_ip_forwarding":false,"ip_configuration":[{"application_gateway_backend_address_pool_ids":null,"application_security_group_ids":null,"load_balancer_backend_address_pool_ids":null,"load_balancer_inbound_nat_rules_ids":null,"name":"internal","primary":true,"public_ip_address":[],"version":"IPv4"}],"name":"example","network_security_group_id":null,"primary":true}],"os_disk":[{"caching":"ReadWrite","diff_disk_settings":[],"disk_encryption_set_id":null,"storage_account_type":"Standard_LRS","write_accelerator_enabled":false}],"overprovision":true,"plan":[],"priority":"Regular","provision_vm_agent":true,"proximity_placement_group_id":null,"resource_group_name":"example-resources","rolling_upgrade_policy":[],"scale_in_policy":"Default","secret":[],"single_placement_group":true,"sku":"Standard_F2","source_image_id":null,"source_image_reference":[{"offer":"UbuntuServer","publisher":"Canonical","sku":"16.04-LTS","version":"latest"}],"tags":null,"timeouts":null,"upgrade_mode":"Manual","zone_balance":false,"zones":null}},{"address":"azurerm_resource_group.example","mode":"managed","type":"azurerm_resource_group","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"location":"westeurope","name":"example-resources","tags":null,"timeouts":null}},{"address":"azurerm_subnet.internal","mode":"managed","type":"azurerm_subnet","name":"internal","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"address_prefixes":["10.0.2.0/24"],"delegation":[],"enforce_private_link_endpoint_network_policies":false,"enforce_private_link_service_network_policies":false,"name":"internal","resource_group_name":"example-resources","service_endpoints":null,"timeouts":null,"virtual_network_name":"example-network"}},{"address":"azurerm_virtual_network.example","mode":"managed","type":"azurerm_virtual_network","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"address_space":["10.0.0.0/16"],"bgp_community":null,"ddos_protection_plan":[],"dns_servers":null,"location":"westeurope","name":"example-network","resource_group_name":"example-resources","tags":null,"timeouts":null,"vm_protection_enabled":false}}]}},"resource_changes":[{"address":"azurerm_linux_virtual_machine_scale_set.example","mode":"managed","type":"azurerm_linux_virtual_machine_scale_set","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"additional_capabilities":[],"admin_password":"P@ssw0rd12345","admin_ssh_key":[],"admin_username":"adminuser","automatic_os_upgrade_policy":[],"boot_diagnostics":[],"custom_data":null,"data_disk":[],"disable_password_authentication":true,"do_not_run_extensions_on_overprovisioned_machines":false,"encryption_at_host_enabled":null,"eviction_policy":null,"health_probe_id":null,"identity":[],"instances":1,"location":"westeurope","max_bid_price":-1,"name":"example-vmss","network_interface":[{"dns_servers":null,"enable_accelerated_networking":false,"enable_ip_forwarding":false,"ip_configuration":[{"application_gateway_backend_address_pool_ids":null,"application_security_group_ids":null,"load_balancer_backend_address_pool_ids":null,"load_balancer_inbound_nat_rules_ids":null,"name":"internal","primary":true,"public_ip_address":[],"version":"IPv4"}],"name":"example","network_security_group_id":null,"primary":true}],"os_disk":[{"caching":"ReadWrite","diff_disk_settings":[],"disk_encryption_set_id":null,"storage_account_type":"Standard_LRS","write_accelerator_enabled":false}],"overprovision":true,"plan":[],"priority":"Regular","provision_vm_agent":true,"proximity_placement_group_id":null,"resource_group_name":"example-resources","rolling_upgrade_policy":[],"scale_in_policy":"Default","secret":[],"single_placement_group":true,"sku":"Standard_F2","source_image_id":null,"source_image_reference":[{"offer":"UbuntuServer","publisher":"Canonical","sku":"16.04-LTS","version":"latest"}],"tags":null,"timeouts":null,"upgrade_mode":"Manual","zone_balance":false,"zones":null},"after_unknown":{"additional_capabilities":[],"admin_ssh_key":[],"automatic_instance_repair":true,"automatic_os_upgrade_policy":[],"boot_diagnostics":[],"computer_name_prefix":true,"data_disk":[],"extension":true,"id":true,"identity":[],"network_interface":[{"ip_configuration":[{"public_ip_address":[],"subnet_id":true}]}],"os_disk":[{"diff_disk_settings":[],"disk_size_gb":true}],"plan":[],"platform_fault_domain_count":true,"rolling_upgrade_policy":[],"secret":[],"source_image_reference":[{}],"terminate_notification":true,"unique_id":true}}},{"address":"azurerm_resource_group.example","mode":"managed","type":"azurerm_resource_group","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"location":"westeurope","name":"example-resources","tags":null,"timeouts":null},"after_unknown":{"id":true}}},{"address":"azurerm_subnet.internal","mode":"managed","type":"azurerm_subnet","name":"internal","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"address_prefixes":["10.0.2.0/24"],"delegation":[],"enforce_private_link_endpoint_network_policies":false,"enforce_private_link_service_network_policies":false,"name":"internal","resource_group_name":"example-resources","service_endpoints":null,"timeouts":null,"virtual_network_name":"example-network"},"after_unknown":{"address_prefix":true,"address_prefixes":[false],"delegation":[],"id":true}}},{"address":"azurerm_virtual_network.example","mode":"managed","type":"azurerm_virtual_network","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"address_space":["10.0.0.0/16"],"bgp_community":null,"ddos_protection_plan":[],"dns_servers":null,"location":"westeurope","name":"example-network","resource_group_name":"example-resources","tags":null,"timeouts":null,"vm_protection_enabled":false},"after_unknown":{"address_space":[false],"ddos_protection_plan":[],"guid":true,"id":true,"subnet":true}}}],"configuration":{"provider_config":{"azurerm":{"name":"azurerm","version_constraint":"=2.37.0","expressions":{"features":[{}]}}},"root_module":{"resources":[{"address":"azurerm_linux_virtual_machine_scale_set.example","mode":"managed","type":"azurerm_linux_virtual_machine_scale_set","name":"example","provider_config_key":"azurerm","expressions":{"admin_password":{"constant_value":"P@ssw0rd12345"},"admin_username":{"constant_value":"adminuser"},"instances":{"constant_value":1},"location":{"references":["azurerm_resource_group.example"]},"name":{"constant_value":"example-vmss"},"network_interface":[{"ip_configuration":[{"name":{"constant_value":"internal"},"primary":{"constant_value":true},"subnet_id":{"references":["azurerm_subnet.internal"]}}],"name":{"constant_value":"example"},"primary":{"constant_value":true}}],"os_disk":[{"caching":{"constant_value":"ReadWrite"},"storage_account_type":{"constant_value":"Standard_LRS"}}],"resource_group_name":{"references":["azurerm_resource_group.example"]},"sku":{"constant_value":"Standard_F2"},"source_image_reference":[{"offer":{"constant_value":"UbuntuServer"},"publisher":{"constant_value":"Canonical"},"sku":{"constant_value":"16.04-LTS"},"version":{"constant_value":"latest"}}]},"schema_version":0},{"address":"azurerm_resource_group.example","mode":"managed","type":"azurerm_resource_group","name":"example","provider_config_key":"azurerm","expressions":{"location":{"constant_value":"West Europe"},"name":{"constant_value":"example-resources"}},"schema_version":0},{"address":"azurerm_subnet.internal","mode":"managed","type":"azurerm_subnet","name":"internal","provider_config_key":"azurerm","expressions":{"address_prefixes":{"constant_value":["10.0.2.0/24"]},"name":{"constant_value":"internal"},"resource_group_name":{"references":["azurerm_resource_group.example"]},"virtual_network_name":{"references":["azurerm_virtual_network.example"]}},"schema_version":0},{"address":"azurerm_virtual_network.example","mode":"managed","type":"azurerm_virtual_network","name":"example","provider_config_key":"azurerm","expressions":{"address_space":{"constant_value":["10.0.0.0/16"]},"location":{"references":["azurerm_resource_group.example"]},"name":{"constant_value":"example-network"},"resource_group_name":{"references":["azurerm_resource_group.example"]}},"schema_version":0}]}}}		`
//			fmt.Println(Handler(context.Background(), events.APIGatewayProxyRequest{Body: something}))
}

func generateErrorResp(ctx context.Context, statusCode int, errorShortCode, errorMessage string) Response {
	resp := Response{
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		StatusCode: statusCode,
	}
	var e errors.APIErrorResp
	seg := xray.GetSegment(ctx)
	if seg != nil {
		e = errors.APIErrorResp{
			StatusCode: statusCode,
			Error:      errorShortCode,
			Message:    errorMessage,
			Suggestion: "File an issue with the TraceId in this payload here: https://github.com/zparnold/azure-terraform-cost-estimator/issues.",
			TraceId:    seg.TraceID,
		}
	}
	e = errors.APIErrorResp{
		StatusCode: statusCode,
		Error:      errorShortCode,
		Message:    errorMessage,
		Suggestion: "File an issue with the TraceId in this payload here: https://github.com/zparnold/azure-terraform-cost-estimator/issues.",
		TraceId:    "notfound",
	}

	b, err := json.Marshal(e)
	if err != nil {
		resp.Body = "There was an error trying to assemble the error"
	} else {
		resp.Body = string(b)
	}
	return resp
}
