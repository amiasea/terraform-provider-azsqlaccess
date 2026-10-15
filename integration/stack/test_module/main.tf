data "azuread_group" "enterprise_sql_test" {
  display_name     = var.entra_group_display_name
  security_enabled = true
}

resource "mssql_azuread_user" "enterprise_sql_test_integration" {
  database_name = var.azure_database_name
  name          = var.entra_group_display_name
  object_id     = data.azuread_group.enterprise_sql_test.object_id

  roles = [
    "db_ddladmin",
    "db_datareader",
    "db_datawriter"
  ]
}