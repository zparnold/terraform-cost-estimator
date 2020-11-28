package pricers

import (
	"fmt"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"strings"
)

type LinuxVM struct {
	Size          string
	Location      string
	Count         float64
	IsSpotEnabled bool
	IsLowPriority bool
}

func (v *LinuxVM) GetHourlyPrice() float64 {
	unitPrice := 0.0
	vms := common.GetItemIfExists(v.GetArn())
	if vms == nil {
		return unitPrice
	}
	for _, vmPriceItem := range *vms {
		if v.IsSpotEnabled && strings.Contains(vmPriceItem.SkuName, "Spot") && !(strings.Contains(vmPriceItem.ProductName, "Windows")) {
			unitPrice = vmPriceItem.UnitPrice
			break
		}
		if v.IsLowPriority && strings.Contains(vmPriceItem.SkuName, "Low Priority") && !(strings.Contains(vmPriceItem.ProductName, "Windows")) {
			unitPrice = vmPriceItem.UnitPrice
			break
		}
		if !v.IsLowPriority && !v.IsSpotEnabled && !(strings.Contains(vmPriceItem.ProductName, "Windows")) {
			unitPrice = vmPriceItem.UnitPrice
			break
		}
	}
	return unitPrice * v.Count
}

func (v *LinuxVM) GetArn() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", "azurerm", "compute", "virtualmachines", strings.ToLower(v.Location), strings.ToLower(v.Size))
}
