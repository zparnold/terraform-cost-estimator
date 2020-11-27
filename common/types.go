package common

import "time"

type ApiResp struct {
	//Future Work
	//PriceItems []ApiRespPriceItem `json:"price_items"`
	EstimatedHourlyCost  float64 `json:"estimated_hourly_cost_usd"`
	EstimatedMonthlyCost float64 `json:"estimated_monthly_cost_usd"`
	EstimatedYearlyCost  float64 `json:"estimated_yearly_cost_usd"`
}
type ApiRespPriceItem struct {
	ResourceType string  `json:"resource_type"`
	Price        float64 `json:"price"`
}

/*

We're making our own statefile because the number of fields we need is very low
*/

type PlanFile struct {
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

type ResourceChange struct {
	Type     string `json:"type"`
	Provider string `json:"provider_name"`
	Change   Change `json:"change"`
}

type Change struct {
	After interface{} `json:"after"`
}

type AzurePricingApiResp struct {
	BillingCurrency    string                `json:"BillingCurrency"`
	CustomerEntityID   string                `json:"CustomerEntityId"`
	CustomerEntityType string                `json:"CustomerEntityType"`
	Items              []AzurePricingApiItem `json:"Items"`
	NextPageLink       *string                `json:"NextPageLink"`
	Count              int                   `json:"Count"`
}

type AzurePricingApiItem struct {
	CurrencyCode         string    `json:"currencyCode"`
	TierMinimumUnits     float64   `json:"tierMinimumUnits"`
	RetailPrice          float64   `json:"retailPrice"`
	UnitPrice            float64   `json:"unitPrice"`
	ArmRegionName        string    `json:"armRegionName"`
	Location             string    `json:"location"`
	EffectiveStartDate   time.Time `json:"effectiveStartDate"`
	MeterID              string    `json:"meterId"`
	MeterName            string    `json:"meterName"`
	ProductID            string    `json:"productId"`
	SkuID                string    `json:"skuId"`
	ProductName          string    `json:"productName"`
	SkuName              string    `json:"skuName"`
	ServiceName          string    `json:"serviceName"`
	ServiceID            string    `json:"serviceId"`
	ServiceFamily        string    `json:"serviceFamily"`
	UnitOfMeasure        string    `json:"unitOfMeasure"`
	Type                 string    `json:"type"`
	IsPrimaryMeterRegion bool      `json:"isPrimaryMeterRegion"`
	ArmSkuName           string    `json:"armSkuName"`
}

