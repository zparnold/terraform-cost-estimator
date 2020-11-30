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
					Count:     change.Change.After.(map[string]interface{})["sku"].([]interface{})[0].(map[string]interface{})["capacity"].(float64),
					Size:      change.Change.After.(map[string]interface{})["sku"].([]interface{})[0].(map[string]interface{})["name"].(string),
					Location:  change.Change.After.(map[string]interface{})["location"].(string),
					//TODO make helpers to grab a key of desired type from chage struct
				})
				break
			case "azurerm_managed_disk":
				resources = append(resources, &azure.AzureDisk{
					Location: change.Change.After.(map[string]interface{})["location"].(string),
					SizeInGb: change.Change.After.(map[string]interface{})["disk_size_gb"].(float64),
					SkuTier:  change.Change.After.(map[string]interface{})["storage_account_type"].(string),
					Count: 1,
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
//		_ = xray.Configure(xray.Config{ContextMissingStrategy: ctxmissing.NewDefaultLogErrorStrategy()})
//		something := `
//{"format_version":"0.1","terraform_version":"0.13.5","planned_values":{"root_module":{"resources":[{"address":"azurerm_managed_disk.main","mode":"managed","type":"azurerm_managed_disk","name":"main","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"create_option":"Empty","disk_encryption_set_id":null,"disk_size_gb":137,"encryption_settings":[],"image_reference_id":null,"location":"westus2","name":"a","os_type":null,"resource_group_name":"a","source_resource_id":null,"storage_account_id":null,"storage_account_type":"UltraSSD_LRS","tags":null,"timeouts":null,"zones":null}},{"address":"azurerm_resource_group.a","mode":"managed","type":"azurerm_resource_group","name":"a","provider_name":"registry.terraform.io/hashicorp/azurerm","schema_version":0,"values":{"location":"westus2","name":"a","tags":null,"timeouts":null}}]}},"resource_changes":[{"address":"azurerm_managed_disk.main","mode":"managed","type":"azurerm_managed_disk","name":"main","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"create_option":"Empty","disk_encryption_set_id":null,"disk_size_gb":137,"encryption_settings":[],"image_reference_id":null,"location":"westus2","name":"a","os_type":null,"resource_group_name":"a","source_resource_id":null,"storage_account_id":null,"storage_account_type":"UltraSSD_LRS","tags":null,"timeouts":null,"zones":null},"after_unknown":{"disk_iops_read_write":true,"disk_mbps_read_write":true,"encryption_settings":[],"id":true,"source_uri":true}}},{"address":"azurerm_resource_group.a","mode":"managed","type":"azurerm_resource_group","name":"a","provider_name":"registry.terraform.io/hashicorp/azurerm","change":{"actions":["create"],"before":null,"after":{"location":"westus2","name":"a","tags":null,"timeouts":null},"after_unknown":{"id":true}}}],"configuration":{"provider_config":{"azurerm":{"name":"azurerm","version_constraint":"=2.37.0","expressions":{"features":[{}]}}},"root_module":{"resources":[{"address":"azurerm_managed_disk.main","mode":"managed","type":"azurerm_managed_disk","name":"main","provider_config_key":"azurerm","expressions":{"create_option":{"constant_value":"Empty"},"disk_size_gb":{"constant_value":137},"location":{"references":["azurerm_resource_group.a"]},"name":{"constant_value":"a"},"resource_group_name":{"references":["azurerm_resource_group.a"]},"storage_account_type":{"constant_value":"UltraSSD_LRS"}},"schema_version":0},{"address":"azurerm_resource_group.a","mode":"managed","type":"azurerm_resource_group","name":"a","provider_config_key":"azurerm","expressions":{"location":{"constant_value":"West US 2"},"name":{"constant_value":"a"}},"schema_version":0}]}}}
//	`
//		fmt.Println(Handler(context.Background(), events.APIGatewayProxyRequest{Body: something}))
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
