package azure

import "context"

type AzureDisk struct {
	SizeInGb int
	Location string
	SkuTier string
}
func (d *AzureDisk) GetHourlyPrice(ctx context.Context) float64 {
return 0.0
}

func (d *AzureDisk) GenerateQuery(ctx context.Context) string {
return ""
}