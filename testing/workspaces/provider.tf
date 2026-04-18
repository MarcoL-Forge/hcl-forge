terraform {
	required_version = ">= 1.5.0"

	required_providers {
		google = {
			source  = "hashicorp/google"
			version = "~> 6.0"
		}

		tfe = {
			source  = "hashicorp/tfe"
			version = "~> 0.64"
		}
	}
}

provider "google" {
	project = var.gcp_service_project_id
	region  = var.gcp_region
}

provider "tfe" {
	hostname = var.tfc_hostname
}
