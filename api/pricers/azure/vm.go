package azure

import (
	"context"
	"fmt"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog/v2"
	"strings"
)

type VirtualMachineAssetPricer struct {
}

func (ap VirtualMachineAssetPricer) Keys() []string {
	return []string{
		"azurerm_windows_virtual_machine",
		"azurerm_linux_virtual_machine",
		"azurerm_virtual_machine_scale_set",
		"azurerm_virtual_machine",
		"azurerm_windows_virtual_machine_scale_set",
		"azurerm_linux_virtual_machine_scale_set",
	}
}

func (ap VirtualMachineAssetPricer) GeneratePricer(change common.ResourceChange) []common.Priceable {
	var isWindows bool
	count := 1.0
	if len(change.Change.After.(map[string]interface{})["os_profile_windows_config"].([]interface{})) > 0 {
		isWindows = true
	}
	if len(change.Change.After.(map[string]interface{})["instances"].([]interface{})) > 0 {
		count = change.Change.After.(map[string]interface{})["instances"].(float64)
	}
	return []common.Priceable{
		&VirtualMachine{
			Size:          change.Change.After.(map[string]interface{})["size"].(string),
			Location:      change.Change.After.(map[string]interface{})["location"].(string),
			Count:         count,
			IsSpotEnabled: change.Change.After.(map[string]interface{})["priority"].(string) == "Spot",
			IsWindows:     isWindows,
		},
	}
}

type VirtualMachine struct {
	IsWindows     bool
	Size          string
	Location      string
	Count         float64
	IsSpotEnabled bool
	IsLowPriority bool
}

func (v *VirtualMachine) GenerateQuery(context.Context) string {
	baseQuery := fmt.Sprintf("serviceName eq 'Virtual Machines' and armRegionName eq '%s' and armSkuName eq '%s' and priceType eq 'Consumption'", v.Location, v.Size)
	var skuFilter []string
	switch {
	case v.IsSpotEnabled:
		skuFilter = append(skuFilter, "contains(skuName, 'Spot')")
		fallthrough
	case v.IsLowPriority:
		skuFilter = append(skuFilter, "contains(skuName, 'Low Priority')")
		fallthrough
	default:
		break
	}
	if v.IsWindows {
		skuFilter = append(skuFilter, "(contains(productName,'Windows') eq true)")
	} else {
		skuFilter = append(skuFilter, "(contains(productName,'Windows') eq false)")
	}
	if !v.IsSpotEnabled && !v.IsLowPriority {
		skuFilter = append(skuFilter, "((contains(skuName,'Spot') eq false) and (contains(skuName,'Low Priority') eq false))")
	}
	//case where we want to rule out spot and low priority
	baseQuery = baseQuery + " and " + strings.Join(skuFilter, " and ")
	return baseQuery
}

func (v *VirtualMachine) GetHourlyPrice(ctx context.Context) float64 {
	unitPrice := 0.0
	vms, err := common.ExecuteAzurePriceQuery(ctx, v)
	if err != nil {
		klog.Error(err)
		return unitPrice
	}
	//Assume that the first one is the one we want
	if len(vms.Items) > 0 {
		unitPrice = vms.Items[0].UnitPrice
	}
	return unitPrice * v.Count
}
