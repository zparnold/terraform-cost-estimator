provider "azurerm" {
  features {}
  version = "=2.37.0"
}

resource "azurerm_resource_group" "a" {
  location = "West US 2"
  name = "a"
}

resource "azurerm_managed_disk" "main" {
  create_option = "Empty"
  location = azurerm_resource_group.a.location
  name = "a"
  resource_group_name = azurerm_resource_group.a.name
  storage_account_type = "UltraSSD_LRS"
  disk_size_gb = 137
}