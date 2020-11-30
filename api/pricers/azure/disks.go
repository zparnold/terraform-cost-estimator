package azure

import (
	"context"
	"fmt"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog/v2"
	"strings"
)

type AzureDisk struct {
	SizeInGb int64
	Location string
	SkuTier string
}
func (v *AzureDisk) GenerateQuery(context.Context) string {
	baseQuery := fmt.Sprintf("serviceName eq 'Storage' and armRegionName eq '%s' and priceType eq 'Consumption'", v.Location)
	var skuFilter []string
	skuApiName := strings.Replace(v.SkuTier,"_", " ", -1)
	skuFilter = append(skuFilter, fmt.Sprintf("contains(skuName, %s)", skuApiName))
	baseQuery = baseQuery + " and " + strings.Join(skuFilter, " and ")
	return baseQuery
}

func (v *AzureDisk) GetHourlyPrice(ctx context.Context) float64 {
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
	// TODO figure out 10k prcing model
	return unitPrice * float64(v.SizeInGb) / 744
}