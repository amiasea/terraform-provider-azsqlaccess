component "test_module" {
  source = "./test_module"

  inputs = {
    entra_group_display_name = var.entra_group_display_name
    azure_database_name      = var.azure_database_name
    azure_database_host      = var.azure_database_host
  }

  providers = {
    azuread = provider.azuread.this
    mssql   = provider.azsqlaccess.this
  }
}
