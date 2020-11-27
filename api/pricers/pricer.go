package pricers

/*
An interface that allows for different ways of fetching a price as long as it represents an estimated hourly cost for
this resource.
*/
type Pricer interface {
	GetHourlyPrice() float64
}
