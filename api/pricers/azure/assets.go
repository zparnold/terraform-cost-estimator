package azure

import (
	""
	"context"
	"github.com/zparnold/azure-terraform-cost-estimator/api/pricers/planfile"
)

func init() {
	ctx := context.Background()
	pricer := planfile.PlanfilePricerInstance(ctx)
	pricer.RegisterAssetPricer(VirtualMachineAssetPricer{})
	pricer.RegisterAssetPricer(AksAssetPricer{})
	pricer.RegisterAssetPricer(ManagedDiskAssetPricer{})
}