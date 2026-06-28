terraform {
  required_version = ">= 1.5.0"
}

resource "google_storage_bucket" "logs" {
  name                        = "legacy-logs-bucket"
  location                    = "europe-west1"
  uniform_bucket_level_access = true
}
