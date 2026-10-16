data "azuread_group" "enterprise_sql_test" {
  display_name     = var.entra_group_display_name
  security_enabled = true
}

resource "azsqlaccess_user" "db_ddladmin" {
  server    = var.azure_database_host
  database  = var.azure_database_name
  type      = "group"
  name      = "db_ddladmin"
  object_id = data.azuread_group.enterprise_sql_test.object_id
}

resource "azsqlaccess_user" "db_datareader" {
  server    = var.azure_database_host
  database  = var.azure_database_name
  type      = "group"
  name      = "db_datareader"
  object_id = data.azuread_group.enterprise_sql_test.object_id
}

resource "azsqlaccess_user" "db_datawriter" {
  server    = var.azure_database_host
  database  = var.azure_database_name
  type      = "group"
  name      = "db_datawriter"
  object_id = data.azuread_group.enterprise_sql_test.object_id
}