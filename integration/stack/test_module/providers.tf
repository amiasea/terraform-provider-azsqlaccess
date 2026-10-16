terraform {
  required_version = ">= 1.16.2"

  required_providers {
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.10"
    }

    mssql = {
      source  = "app.terraform.io/amiasea-development/azsqlaccess"
      version = "1.0.1"
    }
  }
}
