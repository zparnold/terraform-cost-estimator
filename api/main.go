package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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

	price, err := PricePlanFile(request.Body)
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
	r.EstimatedHourlyCost = price
	r.EstimatedMonthlyCost = price * MONTH_HOURS
	r.EstimatedYearlyCost = price * YEAR_HOURS

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
func PricePlanFile(jsonBlob string) (float64, error) {
	var pf common.PlanFile
	err := json.Unmarshal([]byte(jsonBlob), &pf)
	if err != nil {
		klog.Error(err)
		return 0.0, err
	}
	var hourlyPrice float64
	for _, change := range pf.ResourceChanges {
		//we only want to price Azure API changes
		if change.Provider == "registry.terraform.io/hashicorp/azurerm" {
			//Until I find a better way we need to explicitly opt-in price types
			switch change.Type {
			case "azurerm_linux_virtual_machine":
				vm := pricers.VirtualMachine{
					Size:     change.Change.After.(map[string]interface{})["size"].(string),
					Location: change.Change.After.(map[string]interface{})["location"].(string),
				}
				hourlyPrice += vm.GetHourlyPrice()
			default:
				break

			}
		}
	}
	fmt.Println(hourlyPrice)
	return hourlyPrice, nil
}
func main() {
	lambda.Start(Handler)
}


