provider "azurerm" {
  features {}
  version = "=2.37.0"
}

terraform {
  required_version = "0.13.0"
}

resource "azurerm_managed_disk" "main" {
  create_option = ""
  location = ""
  name = ""
  resource_group_name = ""
  storage_account_type = ""
}