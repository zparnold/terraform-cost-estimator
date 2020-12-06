package test

import (
	"context"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/zparnold/azure-terraform-cost-estimator/common/pricers/azure"
	"github.com/zparnold/azure-terraform-cost-estimator/common/types"
	"os"
	"testing"
)

func TestLinuxVmConsumption(t *testing.T) {
	// disable xray when testing locally -otherwise you'll get an x-ray 'segment' error
	_ = os.Setenv("AWS_XRAY_SDK_DISABLED", "true")

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./linuxvmtest/",
		PlanFilePath: "./tfplan.out",
	})

	ctx := context.Background()
	price, unsupportedResources, unestimateableResources, err := azure.PricePlanFile(ctx, jsonPlan, azure.Consumption)
	resp, err := wrapResponse(price, unsupportedResources, unestimateableResources)

	if err != nil {
		t.Error(err)
	}

	assert.InDelta(t, 0.114, resp.TotalEstimate.HourlyCost, 0.0001)
}

func TestLinuxVmReservation(t *testing.T) {
	// disable xray when testing locally -otherwise you'll get an x-ray 'segment' error
	_ = os.Setenv("AWS_XRAY_SDK_DISABLED", "true")

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./linuxvmtest/",
		PlanFilePath: "./tfplan.out",
	})

	ctx := context.Background()
	price, unsupportedResources, unestimateableResources, err := azure.PricePlanFile(ctx, jsonPlan, azure.Reservation1Yr)
	resp, err := wrapResponse(price, unsupportedResources, unestimateableResources)

	if err != nil {
		t.Error(err)
	}
	assert.InDelta(t, 0.0827, resp.TotalEstimate.HourlyCost, 0.0001)

	price, unsupportedResources, unestimateableResources, err = azure.PricePlanFile(ctx, jsonPlan, azure.Reservation3Yr)
	resp, err = wrapResponse(price, unsupportedResources, unestimateableResources)

	if err != nil {
		t.Error(err)
	}
	assert.InDelta(t, 0.0565, resp.TotalEstimate.HourlyCost, 0.0001)
}

func wrapResponse(price float64, unsupportedResources []string, unestimateableResources []string) (types.ApiResp, error) {
	var r types.ApiResp
	r.TotalEstimate.HourlyCost = price
	r.TotalEstimate.MonthlyCost = price * types.MONTH_HOURS
	r.TotalEstimate.YearlyCost = price * types.YEAR_HOURS
	r.UnsupportedResources = unsupportedResources
	r.UnestimateableResources = unestimateableResources
	return r, nil
}
