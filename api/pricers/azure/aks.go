package azure

import (
	"context"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
)

type AksAssetPricer struct {
}

func (ap AksAssetPricer) Keys() []string {
	return []string{
		"azurerm_managed_disk",
	}
}

func (ap AksAssetPricer) GeneratePricer(change common.ResourceChange) []common.Priceable {
	return []common.Priceable{
		&AksCluster{
			IsPaid: change.Change.After.(map[string]interface{})["sku_tier"].(string) == "Paid",
		},
		&VirtualMachine{
			Size:      change.Change.After.(map[string]interface{})["default_node_pool"].([]interface{})[0].(map[string]interface{})["vm_size"].(string),
			Location:  change.Change.After.(map[string]interface{})["location"].(string),
			Count:     change.Change.After.(map[string]interface{})["default_node_pool"].([]interface{})[0].(map[string]interface{})["node_count"].(float64),
			IsWindows: false,
		},
	}
}

type AksCluster struct {
	IsPaid bool
}

func (A *AksCluster) GetHourlyPrice(context.Context) float64 {
	if A.IsPaid {
		return 0.10
	}
	return 0.0
}
