required_providers {
  azuread = {
    source  = "hashicorp/azuread"
    version = "~> 3.10"
  }

  azsqlaccess = {
    source  = "app.terraform.io/amiasea-development/azsqlaccess"
    version = "1.0.3"
  }
}

provider "azuread" "this" {
  config {
    tenant_id = var.azure_tenant_id
    client_id = var.azure_client_id

    oidc_token = var.azure_identity_token

    use_oidc = true
  }
}

provider "azsqlaccess" "this" {
  config {
    engine = "mssql"

    tenant_id = var.azure_tenant_id
    client_id = var.azure_client_id

    oidc_token = var.azure_identity_token
  }
}
