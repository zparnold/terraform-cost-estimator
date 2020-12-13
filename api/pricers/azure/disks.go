package azure

import (
	"context"
	"fmt"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog/v2"
	"strings"
)

const MONTH_HOURS = 730.0

type ManagedDiskAssetPricer struct {
}

func (ap ManagedDiskAssetPricer) Keys() []string {
	return []string{
		"azurerm_managed_disk",
	}
}

func (ap ManagedDiskAssetPricer) GeneratePricer(change common.ResourceChange) []common.Priceable {
	return []common.Priceable{
		&AzureDisk{
			Location: change.Change.After.(map[string]interface{})["location"].(string),
			SizeInGb: change.Change.After.(map[string]interface{})["disk_size_gb"].(float64),
			SkuTier:  change.Change.After.(map[string]interface{})["storage_account_type"].(string),
			Count:    1,
		},
	}
}

type AzureDisk struct {
	SizeInGb float64
	Location string
	SkuTier  string
	Count    int
}

var storageToProductMap = map[string]string{
	"Standard_LRS":    "Standard HDD Managed Disks",
	"StandardSSD_LRS": "Standard SSD Managed Disks",
	"Premium_LRS":     "Premium SSD Managed Disks",
	"UltraSSD_LRS":    "Ultra Disks",
}

func (v *AzureDisk) GenerateQuery(context.Context) string {
	baseQuery := fmt.Sprintf("serviceName eq 'Storage' and armRegionName eq '%s' and priceType eq 'Consumption'", v.Location)
	var skuFilter []string
	skuFilter = append(skuFilter, fmt.Sprintf("productName eq '%s'", storageToProductMap[v.SkuTier]))
	if v.SkuTier != "UltraSSD_LRS" {
		skuFilter = append(skuFilter, fmt.Sprintf("(contains(skuName,'%s') eq true)", v.getDiskSize()))
	} else {
		skuFilter = append(skuFilter, "meterName eq 'Provisioned Capacity'")
	}
	baseQuery = baseQuery + " and " + strings.Join(skuFilter, " and ")
	return baseQuery
}

func (v *AzureDisk) GetHourlyPrice(ctx context.Context) float64 {
	unitPrice := 0.0
	disks, err := common.ExecuteAzurePriceQuery(ctx, v)
	if err != nil {
		klog.Error(err)
		return unitPrice
	}
	//Assume that the first one is the one we want
	if len(disks.Items) > 0 {
		unitPrice = disks.Items[0].UnitPrice
	}
	if v.SkuTier == "UltraSSD_LRS" {
		//This one is metered in per GB
		return unitPrice * v.SizeInGb * float64(v.Count)
	} else {
		//This one is metered by tier
		return (unitPrice * float64(v.Count)) / MONTH_HOURS
	}
}

func (v *AzureDisk) getDiskSize() string {
	switch v.SkuTier {
	case "Standard_LRS":
		return fmt.Sprintf("S%s", getNonUltraDiskNumber(v.SizeInGb))
	case "StandardSSD_LRS":
		return fmt.Sprintf("E%s", getNonUltraDiskNumber(v.SizeInGb))
	case "Premium_LRS":
		return fmt.Sprintf("P%s", getNonUltraDiskNumber(v.SizeInGb))
	default:
		return "unsupported"
	}
}

//https://azure.microsoft.com/en-us/pricing/details/managed-disks/
func getNonUltraDiskNumber(size float64) string {
	switch s := size; {
	case s <= 4.0:
		return "1"
	case s <= 8.0:
		return "2"
	case s <= 16.0:
		return "3"
	case s <= 32.0:
		return "4"
	case s <= 64.0:
		return "6"
	case s <= 128.0:
		return "10"
	case s <= 256.0:
		return "15"
	case s <= 512.0:
		return "20"
	case s <= 1024.0:
		return "30"
	case s <= 2048.0:
		return "40"
	case s <= 4096.0:
		return "50"
	case s <= 8192.0:
		return "60"
	case s <= 16384.0:
		return "70"
	case s <= 32767.0:
		return "80"
	default:
		return "nothing"
	}
}
