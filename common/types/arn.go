package types

import "strings"

func GetArnForAzureApiItem(priceItem *AzurePricingApiItem) string {
	var sb strings.Builder
	sb.WriteString("azurerm")
	sb.WriteString(":")
	sb.WriteString(strings.ToLower(strings.ReplaceAll(priceItem.ServiceFamily, " ", "")))
	sb.WriteString(":")
	sb.WriteString(strings.ToLower(strings.ReplaceAll(priceItem.ServiceName, " ", "")))
	sb.WriteString(":")
	sb.WriteString(strings.ToLower(priceItem.ArmRegionName))
	sb.WriteString(":")
	sb.WriteString(strings.ToLower(priceItem.ArmSkuName))
	return sb.String()
}
