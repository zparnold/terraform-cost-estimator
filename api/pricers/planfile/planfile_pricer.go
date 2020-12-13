package planfile

import (
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"golang.org/x/net/context"
	"sync"
)

var lock = &sync.Mutex{}

type PlanfilePricer struct {
	assetPricers         map[string]*common.TerraformPriceableAsset
	ctx                  context.Context
	unsupportedResources []string
}

var singleton *PlanfilePricer

func PlanfilePricerInstance(ctx context.Context) *PlanfilePricer {
	if singleton == nil {
		lock.Lock()
		defer lock.Unlock()
		if singleton == nil {
			singleton = &PlanfilePricer{
				ctx: ctx,
			}
		}
	}
	return singleton
}

func (planfilePricer *PlanfilePricer) RegisterAssetPricer(asset common.TerraformPriceableAsset) {
	lock.Lock()
	defer lock.Unlock()
	for _, k := range asset.Keys() {
		planfilePricer.assetPricers[k] = &asset
	}
}

func (planfilePricer *PlanfilePricer) GetAssetCost(assetKey string, change common.ResourceChange) (float64, error) {
	tf, exists := planfilePricer.assetPricers[assetKey]
	pricer := *tf
	priceables := pricer.GeneratePricer(change)
	if exists {
		cost := 0.0
		for _,p := range priceables {
			cost = cost + p.GetHourlyPrice(planfilePricer.ctx)
		}
		return cost, nil
	} else {
		return 0.0, common.UnsupportedResourceError{
			Key:     assetKey,
			Address: change.Address,
		}
	}
}
