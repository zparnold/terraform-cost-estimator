package azure

import (
	"context"
	"fmt"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog/v2"
	"strings"
)

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
	if len(vms.Items) > 0{
		unitPrice = vms.Items[0].UnitPrice
	}
	return unitPrice * v.Count
}
