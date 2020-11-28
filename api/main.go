package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/zparnold/azure-terraform-cost-estimator/api/pricers"
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
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	var r common.ApiResp
	var resp Response

	price, unsupportedResources, err := PricePlanFile(request.Body)
	if err != nil {
		resp = Response{
			StatusCode:      500,
			IsBase64Encoded: false,
			Body:            fmt.Sprintf("{\"error\":\"%s\"", err.Error()),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}
	}
	r.TotalEstimate.HourlyCost = price
	r.TotalEstimate.MonthlyCost = price * MONTH_HOURS
	r.TotalEstimate.YearlyCost = price * YEAR_HOURS
	r.UnsupportedResources = unsupportedResources
	b, err := json.Marshal(r)
	if err != nil {
		resp = Response{
			StatusCode:      500,
			IsBase64Encoded: false,
			Body:            fmt.Sprintf("{\"error\":\"%s\"", err.Error()),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}
	}
	resp = Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(b),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return resp, nil
}
func PricePlanFile(jsonBlob string) (float64, []string, error) {
	var unsupportedResources []string
	var pf common.PlanFile
	err := json.Unmarshal([]byte(jsonBlob), &pf)
	if err != nil {
		klog.Error(err)
		return 0.0, []string{}, err
	}
	var hourlyPrice float64
	var resources []pricers.Pricer

	for _, change := range pf.ResourceChanges {
		//we only want to price Azure API changes
		if change.Provider == "registry.terraform.io/hashicorp/azurerm" {
			//Until I find a better way we need to explicitly opt-in price types
			switch change.Type {
			case "azurerm_linux_virtual_machine":
				resources = append(resources, &pricers.LinuxVM{
					Size:     change.Change.After.(map[string]interface{})["size"].(string),
					Location: change.Change.After.(map[string]interface{})["location"].(string),
					Count: 1.0,
					IsSpotEnabled: change.Change.After.(map[string]interface{})["priority"].(string) == "Spot",
				})
			case "azurerm_windows_virtual_machine":
				resources = append(resources, &pricers.WindowsVM{
					Size:     change.Change.After.(map[string]interface{})["size"].(string),
					Location: change.Change.After.(map[string]interface{})["location"].(string),
					Count:  1.0,
					IsSpotEnabled: change.Change.After.(map[string]interface{})["priority"].(string) == "Spot",
				})
			case "azurerm_kubernetes_cluster":
				fmt.Println(change.Change.After.(map[string]interface{})["default_node_pool"].([]interface{})[0].(map[string]interface{})["node_count"].(float64))
				resources = append(resources, &pricers.AksCluster{
				IsPaid: change.Change.After.(map[string]interface{})["sku_tier"].(string) == "Paid",
				},
				&pricers.LinuxVM{
					Size:     change.Change.After.(map[string]interface{})["default_node_pool"].([]interface{})[0].(map[string]interface{})["vm_size"].(string),
					Location: change.Change.After.(map[string]interface{})["location"].(string),
					Count: change.Change.After.(map[string]interface{})["default_node_pool"].([]interface{})[0].(map[string]interface{})["node_count"].(float64),
				})
			//This is where a resource that is unsupported	will fall through
			default:
				unsupportedResources = append(unsupportedResources, change.Address)
				break
			}
		}
	}

	for _, res := range resources {
		hourlyPrice += res.GetHourlyPrice()
	}

	return hourlyPrice, unsupportedResources, nil
}

func main() {
	//lambda.Start(Handler)
	resp, _ := Handler(context.Background(), events.APIGatewayProxyRequest{Body:
		`
{"format_version":"0.1","terraform_version":"0.13.0","planned_values":{"outputs":{"client_certificate":{"sensitive":false},"kube_config":{"sensitive":false}},"root_module":{"resources":[{"address":"azurerm_kubernetes_cluster.example","mode":"managed","type":"azurerm_kubernetes_cluster","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"api_server_authorized_ip_ranges":null,"default_node_pool":[{"availability_zones":null,"enable_auto_scaling":null,"enable_node_public_ip":null,"max_count":null,"min_count":null,"name":"default","node_count":5,"node_labels":null,"node_taints":null,"os_disk_type":"Managed","proximity_placement_group_id":null,"tags":null,"type":"VirtualMachineScaleSets","vm_size":"Standard_D2_v2","vnet_subnet_id":null}],"disk_encryption_set_id":null,"dns_prefix":"exampleaks1","enable_pod_security_policy":null,"identity":[{"type":"SystemAssigned"}],"linux_profile":[],"location":"westeurope","name":"example-aks1","resource_group_name":"example-resources","service_principal":[],"sku_tier":"Paid","tags":{"Environment":"Production"},"timeouts":null}},{"address":"azurerm_resource_group.example","mode":"managed","type":"azurerm_resource_group","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"location":"westeurope","name":"example-resources","tags":null,"timeouts":null}}]}},"resource_changes":[{"address":"azurerm_kubernetes_cluster.example","mode":"managed","type":"azurerm_kubernetes_cluster","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"api_server_authorized_ip_ranges":null,"default_node_pool":[{"availability_zones":null,"enable_auto_scaling":null,"enable_node_public_ip":null,"max_count":null,"min_count":null,"name":"default","node_count":5,"node_labels":null,"node_taints":null,"os_disk_type":"Managed","proximity_placement_group_id":null,"tags":null,"type":"VirtualMachineScaleSets","vm_size":"Standard_D2_v2","vnet_subnet_id":null}],"disk_encryption_set_id":null,"dns_prefix":"exampleaks1","enable_pod_security_policy":null,"identity":[{"type":"SystemAssigned"}],"linux_profile":[],"location":"westeurope","name":"example-aks1","resource_group_name":"example-resources","service_principal":[],"sku_tier":"Paid","tags":{"Environment":"Production"},"timeouts":null},"after_unknown":{"addon_profile":true,"auto_scaler_profile":true,"default_node_pool":[{"max_pods":true,"orchestrator_version":true,"os_disk_size_gb":true}],"fqdn":true,"id":true,"identity":[{"principal_id":true,"tenant_id":true}],"kube_admin_config":true,"kube_admin_config_raw":true,"kube_config":true,"kube_config_raw":true,"kubelet_identity":true,"kubernetes_version":true,"linux_profile":[],"network_profile":true,"node_resource_group":true,"private_cluster_enabled":true,"private_fqdn":true,"private_link_enabled":true,"role_based_access_control":true,"service_principal":[],"tags":{},"windows_profile":true}}},{"address":"azurerm_resource_group.example","mode":"managed","type":"azurerm_resource_group","name":"example","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"location":"westeurope","name":"example-resources","tags":null,"timeouts":null},"after_unknown":{"id":true}}}],"output_changes":{"client_certificate":{"actions":["create"],"before":null,"after_unknown":true},"kube_config":{"actions":["create"],"before":null,"after_unknown":true}},"configuration":{"provider_config":{"azurerm":{"name":"azurerm","version_constraint":"=2.37.0","expressions":{"features":[{}]}}},"root_module":{"outputs":{"client_certificate":{"expression":{"references":["azurerm_kubernetes_cluster.example"]}},"kube_config":{"expression":{"references":["azurerm_kubernetes_cluster.example"]}}},"resources":[{"address":"azurerm_kubernetes_cluster.example","mode":"managed","type":"azurerm_kubernetes_cluster","name":"example","provider_config_key":"azurerm","expressions":{"default_node_pool":[{"name":{"constant_value":"default"},"node_count":{"constant_value":5},"vm_size":{"constant_value":"Standard_D2_v2"}}],"dns_prefix":{"constant_value":"exampleaks1"},"identity":[{"type":{"constant_value":"SystemAssigned"}}],"location":{"references":["azurerm_resource_group.example"]},"name":{"constant_value":"example-aks1"},"resource_group_name":{"references":["azurerm_resource_group.example"]},"sku_tier":{"constant_value":"Paid"},"tags":{"constant_value":{"Environment":"Production"}}},"schema_version":0},{"address":"azurerm_resource_group.example","mode":"managed","type":"azurerm_resource_group","name":"example","provider_config_key":"azurerm","expressions":{"location":{"constant_value":"West Europe"},"name":{"constant_value":"example-resources"}},"schema_version":0}]}}}

`,
		})
	fmt.Println(resp.Body)

}
