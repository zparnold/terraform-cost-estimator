package azure

import "context"

type AksCluster struct {
	IsPaid bool
}
func (A *AksCluster) GetHourlyPrice(context.Context) float64 {
	if A.IsPaid {
		return 0.10
	}
	return 0.0
}
