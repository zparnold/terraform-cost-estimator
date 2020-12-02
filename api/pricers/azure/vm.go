package azure

import (
	"context"
	"errors"
	"fmt"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"k8s.io/klog/v2"
	"strings"
)

type PriceType int

const (
	Consumption PriceType = iota
	DevTestConsumption
	Reservation1Yr
	Reservation3Yr
)

var PriceTypeLookup = map[string]PriceType{
	"consumption":    Consumption,
	"reservation1yr": Reservation1Yr,
	"reservation3yr": Reservation3Yr,
}

type VirtualMachine struct {
	IsWindows     bool
	Size          string
	Location      string
	Count         float64
	IsSpotEnabled bool
	IsLowPriority bool
	PriceType     PriceType
}

func (v *VirtualMachine) GenerateQuery(context.Context) string {
	baseQuery := fmt.Sprintf("serviceName eq 'Virtual Machines' and armRegionName eq '%s' and armSkuName eq '%s'", v.Location, v.Size)
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

	if useReservationBilling(*v) {
		skuFilter = append(skuFilter, "priceType eq 'Reservation'")
	} else if v.PriceType == Consumption {
		skuFilter = append(skuFilter, "priceType eq 'Consumption'")
	} else if v.PriceType == DevTestConsumption {
		skuFilter = append(skuFilter, "priceType eq 'DevTestConsumption'")
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
		//The unitPrice reflects annual amounts for Reservation instances.  Need to convert this to an hourlyRate
		if useReservationBilling(*v) {
			for _, item := range vms.Items {
				//we can't filter on 'reservationTerm' in the ODATA query, so we need to do it here
				if item.ReservationTerm == "1 Year" && v.PriceType == Reservation1Yr {
					unitPrice = item.UnitPrice / common.YEAR_HOURS
					break
				} else if item.ReservationTerm == "3 Years" && v.PriceType == Reservation3Yr {
					unitPrice = item.UnitPrice / (3.0 * common.YEAR_HOURS)
					break
				}
			}
			if unitPrice == 0.0 {
				//We couldn't find a match.  Do we need to return this error or simply log it
				err = errors.New(fmt.Sprintf("Could not find an item with reservation duration %d", v.PriceType))
				klog.Error(err)

			}
		} else {
			unitPrice = vms.Items[0].UnitPrice
		}
	}

	return unitPrice * v.Count
}

func useReservationBilling(v VirtualMachine) bool {
	if v.PriceType == Reservation1Yr || v.PriceType == Reservation3Yr {
		return true
	}
	return false
}
