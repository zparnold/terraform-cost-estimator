# Terraform Cost Estimator

Helps you to estimate costs of Terraform Plans and right now is focused on the [`azurerm`](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs) 
Terraform Provider. _Note these are **estimates only** (not actual costs)
based on "Pay-as-you-Go" pricing. The point of this API is only so you have an estimate, not a guarantee of potential future costs._ 
If you're looking for other pricing schemes, such as "reserved", "DevTest", or if your company has an agreement
with a cloud provider that gives you a discount off of standard "Pay-as-you-Go" list prices this will not be reflected here. Hopefully,
if you are looking for one of these pricing schemes, this API should at least provide you with an upper-bound. However,
it in no way represents a guarantee of prices between you and your cloud provider.

### Can I use this for more cloud providers than Azure?
It is definitely designed to provide this functionality, but at present since I work at a company that uses Azure I'm
focused on that. That being said, PR's welcome!

## Usage
**One more time for emphasis, this is only an estimate of expected future cloud costs!**
```bash
terraform plan -out=plan.tfplan > /dev/null && terraform show -json plan.tfplan  | curl -s -X POST -H "Content-Type: application/json" -d @- https://api-dev.pricing.tf/estimate
```
### Response
```json
{
    "unsupported_resources": [
        "azurerm_network_interface.example"
    ],
    "unestimateable_resources": [
        "azurerm_resource_group.example",
        "azurerm_subnet.example",
        "azurerm_virtual_network.example"
    ],
    "estimate_summary": {
        "hourly_cost_usd": 0.114,
        "monthly_cost_usd": 83.22,
        "yearly_cost_usd": 998.64
    }
}
```
The response provides:
* `unsupported_resources` to let you know which resources weren't priced
* `estimate_summary` which contains the Hourly, Monthly, and Yearly additional cost based on this Terraform plan
* `unestimateable_resources` to let you know which resources are not currently able to be estimated based on this terraform plan

_Note: currently "monthly" and "yearly" prices are only calculated as a multiple of hours. 1 Month = 730 Hours and 1 Year = 8760 Hours._

## Security
The code is all here and executes in a serverless function, you can read for yourself and see that we're not storing/logging anything
you send. :smile:

## Supported Resources
||Resource Name|Area|
|---|---|---|
|[x]|`azurerm_linux_virtual_machine`|Compute|
|[x]|`azurerm_windows_virtual_machine`|Compute|
|[x]|`azurerm_virutal_machine`|Compute|
|[x]|`azurerm_virutal_machine_scale_set`|Compute|
|[x]|`azurerm_linux_virutal_machine_scale_set`|Compute|
|[x]|`azurerm_windows_virutal_machine_scale_set`|Compute|
|[x]|`azurerm_kubernetes_cluster`|Containers|

#### A side note on billable units of measure:
Not all billable resources in Azure are tied to an hourly price. For example, consider VNETs/egress, StorageAccount Blob Storage consumed size,
or anything tied to API call count like KeyVault. These resources depend on further consumption after provisioning, so they
are in-effect unestimateable from the standpoint of this API. In theory, one could derive an estimate based on average consumption across all Azure usage,
but I don't work for Microsoft/nor have access to that data. So, they will probably remain unestimated unless you have a 
good idea and want to contribute!

## To Dev
* Ensure that go >= 1.13 and `serverless` 2.x is installed on your machine
* Make a PR (add a test too)
* Assign me (zparnold) and I will try to get it merged and deployed.

## Adding Resources to Price
The `api/pricers/` folder is where a collection of interfaces of type `Pricer` are implemented. The only function necessary to
implement this interface is `GetHourlyPrice()` which returns a `float64`. Should you want to use the existing price database (Dynamo)
take a look at the resource enumeration section below. There is also a `prices.csv` that should give you a full list of 
priceable objects in Azure returned by their public API here: https://docs.microsoft.com/en-us/rest/api/cost-management/retail-prices/azure-retail-prices

To add another resource to be priced:

* Add a new file, preferably in the `api/pricers/` folder using a name that would help others understand it
* Implement the function from above
* Add a `case` statement in the `api/main.go` which binds the terraform resource name to the pricer (copy one from the method.)

## Runtime Cloud Design
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
one item. See an example `api/pricers/windows_vm.go`

## The path to 1.0 (and prod)
|Status|Task|
|---|---|
|[x]|Integration tests exist|
||Integration tests in CI pipeline|
||Automated deployment in CI Pipeline|
|[x]|Basic compute resources supported|
||Basic storage resources supported|
||Estimateable networking resources supported|
||Some PaaS or SaaS resources supported maybe? (Azure App Services, Redis, Azure Functions, AKS, ACI)|