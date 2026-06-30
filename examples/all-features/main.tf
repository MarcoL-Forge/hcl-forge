terraform {
  required_version = ">= 1.5.0"
}

module "service-account-123" {
  source     = "./modules/service-account"
  project_id = "legacy-project"
}

module "service-account-456" {
  source     = "./modules/service-account"
  project_id = "legacy-project"
}

module "network" {
  source = "./modules/network"
}

resource "google_storage_bucket" "logs" {
  name                        = "legacy-logs-bucket"
  location                    = "europe-west1"
  uniform_bucket_level_access = true
  enable_secure_boot          = true
  enable_integrity_monitoring = true
  tags                        = ["legacy-node"]
}

resource "google_storage_bucket" "archive" {
  name     = "legacy-archive-bucket"
  location = "us-central1"
}

resource "google_container_node_pool" "pool" {
  name = "primary-pool"

  node_config {
    machine_type = "e2-medium"
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  lifecycle {
    ignore_changes = [name]
  }

  lifecycle {
    ignore_changes = [version]
  }

  tags = ["legacy-node"]
}

variable "project_id" {
  type = string
}
