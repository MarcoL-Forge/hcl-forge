locals {
	workspace_service_accounts = {
		gke = {
			account_id   = var.gke_workspace_service_account_id
			display_name = "Terraform GKE workspace"
			description  = "Service account used by the Terraform workspace that manages GKE."
		}
		service_project = {
			account_id   = var.service_project_workspace_service_account_id
			display_name = "Terraform service project workspace"
			description  = "Service account used by the Terraform workspace that manages the GCP service project."
		}
	}
}

resource "google_service_account" "workspace" {
	for_each = local.workspace_service_accounts

	project      = var.gcp_service_project_id
	account_id   = each.value.account_id
	display_name = each.value.display_name
	description  = each.value.description
}

resource "google_project_iam_member" "gke_service_project" {
	for_each = toset(var.gke_workspace_roles)

	project = var.gcp_service_project_id
	role    = each.value
	member  = "serviceAccount:${google_service_account.workspace["gke"].email}"
}

resource "google_project_iam_member" "gke_host_project" {
	for_each = toset(var.gke_workspace_host_project_roles)

	project = var.gcp_host_project_id
	role    = each.value
	member  = "serviceAccount:${google_service_account.workspace["gke"].email}"
}

resource "google_project_iam_member" "service_project_workspace" {
	for_each = toset(var.service_project_workspace_roles)

	project = var.gcp_service_project_id
	role    = each.value
	member  = "serviceAccount:${google_service_account.workspace["service_project"].email}"
}
