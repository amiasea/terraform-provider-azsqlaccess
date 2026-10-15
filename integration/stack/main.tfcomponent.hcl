component "test_module" {
  source = "./test_module"

  inputs = {
    entra_group_display_name = var.entra_group_display_name
    azure_database_name      = var.azure_database_name
  }

  providers = {
    azuread = provider.azuread.this
    mssql   = provider.mssql.this
  }
}