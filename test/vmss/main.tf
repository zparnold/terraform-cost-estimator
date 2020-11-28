provider "azurerm" {
  features {}
  version = "=2.37.0"
}

terraform {
  required_version = "0.13.0"
}

resource "azurerm_resource_group" "example" {
  name = "example-resources"
  location = "West Europe"
}

resource "azurerm_windows_virtual_machine_scale_set" "example" {
  admin_password = ""
  admin_username = ""
  instances = 0
  location = azurerm_resource_group.example.location
  name = ""
  resource_group_name = azurerm_resource_group.example.name
  sku = "Standard_E48_v4"
  network_interface {
    name = ""
    ip_configuration {
      name = ""
    }
  }
  os_disk {
    caching = ""
    storage_account_type = ""
  }
}