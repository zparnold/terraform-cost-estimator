provider "azurerm" {
  features {}
  version = "=2.37.0"
}

resource "azurerm_resource_group" "example" {
  name = "example-resources"
  location = "West Europe"
}

resource "azurerm_kubernetes_cluster" "example" {
  name = "example-aks1"
  location = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  dns_prefix = "exampleaks1"
  sku_tier = "Paid"
  default_node_pool {
    name = "default"
    node_count = 5
    vm_size = "Standard_D2_v2"
  }

  identity {
    type = "SystemAssigned"
  }

  tags = {
    Environment = "Production"
  }
}

output "client_certificate" {
  value = azurerm_kubernetes_cluster.example.kube_config.0.client_certificate
}

output "kube_config" {
  value = azurerm_kubernetes_cluster.example.kube_config_raw
}