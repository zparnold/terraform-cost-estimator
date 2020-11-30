package test

import (
	"bytes"
	"encoding/json"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	REST_ENDPOINT = "https://api-dev.pricing.tf/estimate"
)

func ExecutePricingOp(jsonPlan string) (*common.ApiResp, error) {
	//runs on existing json serialized plan, so no need to marshal into json
	response, err := http.Post(REST_ENDPOINT, "application/json", bytes.NewBufferString(jsonPlan))
	if err != nil || response.StatusCode != 200 {
		return nil, err
	}
	var responseObj common.ApiResp
	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(respBody, &responseObj)
	if err != nil {
		return nil, err
	}
	return &responseObj, nil
}

func TestLinuxVm(t *testing.T) {

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./linuxvmtest/",
		PlanFilePath: "./tfplan.out",
	})
	resp, err := ExecutePricingOp(jsonPlan)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 0.114, resp.TotalEstimate.HourlyCost)
}
func TestWindowsVmWithLicenseCost(t *testing.T) {

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./windowsvmtest/",
		PlanFilePath: "./tfplan.out",
	})
	resp, err := ExecutePricingOp(jsonPlan)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 0.206, resp.TotalEstimate.HourlyCost)
}

func TestAksClusterWith5Nodes(t *testing.T) {

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./aks/",
		PlanFilePath: "./tfplan.out",
	})
	resp, err := ExecutePricingOp(jsonPlan)
	if err != nil {
		t.Error(err)
	}
	expectedPrice := (0.136 * 5) + 0.10 // aks control plane paid sku
	assert.Equal(t, expectedPrice, resp.TotalEstimate.HourlyCost)
}

func TestVmss(t *testing.T) {

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./vmss/",
		PlanFilePath: "./tfplan.out",
	})
	resp, err := ExecutePricingOp(jsonPlan)
	if err != nil {
		t.Error(err)
	}
	expectedPrice := (0.099 * 2)
	assert.Equal(t, expectedPrice, resp.TotalEstimate.HourlyCost)
}

func TestLegacyVmResource(t *testing.T) {

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./legacyvm/",
		PlanFilePath: "./tfplan.out",
	})
	resp, err := ExecutePricingOp(jsonPlan)
	if err != nil {
		t.Error(err)
	}
	expectedPrice := 0.198
	assert.Equal(t, expectedPrice, resp.TotalEstimate.HourlyCost)
}
func TestWindowsVmss(t *testing.T) {

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./windows_vmss/",
		PlanFilePath: "./tfplan.out",
	})
	resp, err := ExecutePricingOp(jsonPlan)
	if err != nil {
		t.Error(err)
	}
	expectedPrice := 0.206
	assert.Equal(t, expectedPrice, resp.TotalEstimate.HourlyCost)
}
func TestLinuxVmss(t *testing.T) {

	jsonPlan := terraform.InitAndPlanAndShow(t, &terraform.Options{
		TerraformDir: "./linux_vmss/",
		PlanFilePath: "./tfplan.out",
	})
	resp, err := ExecutePricingOp(jsonPlan)
	if err != nil {
		t.Error(err)
	}
	expectedPrice := 0.114
	assert.Equal(t, expectedPrice, resp.TotalEstimate.HourlyCost)
}
