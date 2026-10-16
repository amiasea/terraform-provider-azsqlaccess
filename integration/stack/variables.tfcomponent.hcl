variable "azure_tenant_id" {
  type = string
}

variable "azure_client_id" {
  type = string
}

variable "entra_group_display_name" {
  type = string
}

variable "azure_database_host" {
  type = string
}

variable "azure_database_name" {
  type = string
}

variable "azure_identity_token" {
  type        = string
  description = "OIDC identity token used to authenticate to Azure."
  ephemeral   = true
}
