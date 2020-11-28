# Terraform Cost Estimator (right now just for Azure)

Helps to estimate costs of Terraform Plans for the AzureRM Terraform Provider

## Usage
```bash
terraform plan -out=plan.tfplan > /dev/null && terraform show -json plan.tfplan  | curl -s -X POST -H "Content-Type: application/json" -d @- https://api.pricing.tf/estimate
```
### Response
```json
{
    "unsupported_resources": [
        "azurerm_network_interface.example",
        "azurerm_resource_group.example",
        "azurerm_subnet.example",
        "azurerm_virtual_network.example"
    ],
    "summary": {
        "hourly_cost_usd": 0.114,
        "monthly_cost_usd": 83.22,
        "yearly_cost_usd": 998.64
    }
}
```
The response provides:
* `unsupported_resources` to let you know which resources weren't priced
* `summary` which contains the Hourly, Monthly, and Yearly additional cost based on this Terraform plan

_Note: currently "monthly" and "yearly" prices are only calculated as a multiple of hours. 1 Month = 730 Hours and 1 Year = 8760 Hours._

## Security
The code is all here and executes in an AWS lambda function, you can read for yourself and see that we're not storing anything
you send. :smile:

## Supported Resources
||Resource Name|
|---|---|
|[x]|`azurerm_linux_virtual_machine`|
|[x]|`azurerm_windows_virtual_machine`|
|[x]|`azurerm_kubernetes_cluster`|

## To Dev
* Ensure that go >= 1.13 and `serverless` 2.x is installed on your machine
* Make a PR (add a test too)
* Assign me (zparnold) and I will try to get it merged and deployed.

## Adding Resources to Price
The `api/pricers/` folder is where a collection of interfaces of type `Pricer` are implemented. The only function necessary to
implement this interface is `GetHourlyPrice()` which returns. Should you want to use the existing price database (Dynamo)
take a look at the resource enumeration section below. There is also a prices.csv that should give you a full list of 
priceable objects in Azure returned by their public API here: https://docs.microsoft.com/en-us/rest/api/cost-management/retail-prices/azure-retail-prices

To add another resource to be priced:

* Add a new file, preferably in the `api/pricers/` folder using a name that would help others understand it
* Implement the function from above
* Add a `case` statement in the `api/main.go` which binds the terraform resource name to the pricer (copy one from the method.)

## Design
![Terraform Cost Estimation Architecture Diagram](./assets/Terraform%20Cost%20Estimator%20Design.png)

## Resource Enumeration
I came up with an ARN-like syntax to uniquely identify product family/sku combinations for pricing, this was for two reasons:
* While the `MeterId` uniquely identifies a billable asset in Azure, terraform state files do not contain this information
* I needed a quick composite primary key for Dynamo so that I can quickly fetch only the correct item instead of complex query/filter logic

### Fields
Each portion (like an ARN) is separated with a colon
* All `id`s begin with the name of the terraform provider for extensibility later: (ex: `azurerm`)
* The next portion is the service family (ex: `compute`)
* Then the service name (responses from azure api are downcased and spaces are removed)(ex: `virtualmachines`)
* Then the region (ex: `westus`)
* Then the ARM sku (ex: `standard_a1_v2`)

If this still does not uniquely identify a billable asset (because it has multiple variants such as Windows/Spot/Low Priority)
then the document associated with this row in Dynamo will have more than one price item in it. In this case it is the
responsibility of the instance of the `Pricer` interface to implement the logic which will further reduce the option to 
one item.