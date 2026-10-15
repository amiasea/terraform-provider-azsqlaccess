store "varset" "integration" {
  name     = "github-actions-integration"
  category = "terraform"
}

identity_token "azure" {
  audience = ["api://AzureADTokenExchange"]
}

deployment "integration_test" {
  inputs = {
    azure_tenant_id          = store.varset.integration.azure_tenant_id
    azure_client_id          = store.varset.integration.azure_client_id
    azure_subscription_id    = store.varset.integration.azure_subscription_id
    azure_identity_token     = identity_token.azure.jwt
    azure_database_host      = store.varset.integration.azure_database_host
    azure_database_name      = store.varset.integration.azure_database_name
    # database_engine          = store.varset.integration.database_engine
    entra_group_display_name = store.varset.integration.entra_group_display_name
  }
}