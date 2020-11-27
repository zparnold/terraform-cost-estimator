package pricers

type AksCluster struct {
	IsPaid bool
}
func (A *AksCluster) GetHourlyPrice() float64 {
	if A.IsPaid {
		return 0.10
	}
	return 0.0
}
