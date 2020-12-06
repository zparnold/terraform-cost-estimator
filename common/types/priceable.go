package types

import "context"

/*
An interface that allows for different ways of fetching a price as long as it represents an estimated hourly cost for
this resource.
*/
type Priceable interface {
	GetHourlyPrice(ctx context.Context) float64
}
