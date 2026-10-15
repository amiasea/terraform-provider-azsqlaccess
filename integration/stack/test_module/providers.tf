terraform {
  required_version = ">= 1.16.2"

  required_providers {
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.10"
    }

    mssql = {
      source  = "MewsSystems/azsqlaccess"
      version = "~> 1.0.0"
    }
  }
}
