package pricers

import (
	"fmt"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog"
	"strings"
)

type WindowsVM struct {
	Size          string
	Location      string
	Count         float64
	IsSpotEnabled bool
	IsLowPriority bool
}

func (v *WindowsVM) GetHourlyPrice() float64 {
	unitPrice := 0.0
	vms, err := common.ExecuteAzurePriceQuery(v)
	if err != nil {
		klog.Error(err)
		return unitPrice
	}
	//Assume that the first one is the one we want
	unitPrice = vms.Items[0].UnitPrice
	return unitPrice * v.Count
}

func (v *WindowsVM) GenerateQuery() string {
	baseQuery := fmt.Sprintf("serviceName eq 'Virtual Machines' and armRegionName eq '%s' and armSkuName eq '%s' and priceType eq 'Consumption' and (contains(productName,'Windows') eq true)", v.Location, v.Size)
	var skuFilter []string
	switch {
	case v.IsSpotEnabled:
		skuFilter = append(skuFilter, "contains(skuName, 'Spot')")
		break
	case v.IsLowPriority:
		skuFilter = append(skuFilter, "contains(skuName, 'Low Priority')")
		break
	default:
		break
	}
	//case where we want to rule out spot and low priority
	if len(skuFilter) == 0 {
		baseQuery = baseQuery + " and ((contains(skuName,'Spot') eq false) and (contains(skuName,'Low Priority') eq false))"
	} else {
		baseQuery = baseQuery + " and " + strings.Join(skuFilter, " and ")
	}
	return baseQuery
}

func (v *WindowsVM) GetArn() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", "azurerm", "compute", "virtualmachines", strings.ToLower(v.Location), strings.ToLower(v.Size))
}
