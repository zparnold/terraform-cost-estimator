package common

//This interface has a function that returns a query that can be run against the Azure pricing API
type AzurePriceableAsset interface {
	GenerateQuery() string
}