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
  name              = "example-workspace1"
  organization      = "example-org"
  execution_mode    = "remote"
  tag_names         = ["hcl-forge", "example"]
  terraform_version = "1.9.8"
}


module "tfe_workspace" "example3" {
  name              = "example-workspace3"
  organization      = "example-org"
  execution_mode    = "remote"
  tag_names         = ["hcl-forge", "example"]
  terraform_version = "1.9.8"

}
