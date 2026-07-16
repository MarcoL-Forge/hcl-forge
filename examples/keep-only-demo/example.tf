terraform {
  required_providers {
    tfe = {
      source  = "hashicorp/tfe"
      version = "~> 0.64"
    }
  }
}

provider "tfe" {}

module "tfe_workspace" "example1" {
  name              = "example-rtl-int-workspace1-gke01"
  organization      = "example-org"
  execution_mode    = "remote"
  tag_names         = ["hcl-forge", "example"]
  terraform_version = "1.9.8"
}

module "tfe_workspace" "example2" {
  name              = "example-rtl-int-workspace2-prj"
  organization      = "example-org"
  execution_mode    = "remote"
  tag_names         = ["hcl-forge", "example"]
  terraform_version = "1.9.8"
}

module "tfe_workspace" "example3" {
  name              = "example-rtl-int-workspace3-sm"
  organization      = "example-org"
  execution_mode    = "remote"
  tag_names         = ["hcl-forge", "example"]
  terraform_version = "1.9.8"

  provider "tfe" {
    alias = "example_tfe3"
  }
}
