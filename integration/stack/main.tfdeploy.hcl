store "varset" "integration" {
  name     = "github-actions-integration"
  category = "terraform"
}

identity_token "azure" {
  audience = ["api://AzureADTokenExchange"]
}


deployment "integration_test" {
  inputs = {
    azure_tenant_id          = store.varset.integration.stable.azure_tenant_id
    azure_client_id           = store.varset.integration.stable.azure_client_id
    azure_identity_token      = identity_token.azure.jwt
    azure_database_host       = store.varset.integration.stable.azure_database_host
    azure_database_name       = store.varset.integration.stable.azure_database_name
    # database_engine         = store.varset.integration.stable.database_engine
    entra_group_display_name  = store.varset.integration.stable.entra_group_display_name
  }
}
