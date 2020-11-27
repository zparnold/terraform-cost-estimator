# azure-terraform-cost-estimator
Helps to estimate costs of Terraform Plans for the AzureRM Terraform Provider

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