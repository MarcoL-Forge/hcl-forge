locals {
	workspace_configs = {
		gke = {
			workspace_name     = var.gke_workspace_name
			service_account_id = var.gke_workspace_service_account_id
			display_name       = "Terraform GKE workspace"
			description        = "Service account used by the Terraform workspace that manages GKE."
			roles              = var.gke_workspace_roles
		}
		service_project = {
			workspace_name     = var.service_project_workspace_name
			service_account_id = var.service_project_workspace_service_account_id
			display_name       = "Terraform service project workspace"
			description        = "Service account used by the Terraform workspace that manages the GCP service project."
			roles              = var.service_project_workspace_roles
		}
	}

	workspace_role_bindings = merge(
		[
			for workspace_key, workspace_config in local.workspace_configs : {
				for role in workspace_config.roles : "${workspace_key}:${role}" => {
					workspace_key = workspace_key
					role          = role
				}
			}
		]...
	)

	gke_cluster_service_account_bindings = {
		for role in var.gke_cluster_service_account_user_roles : role => role
	}

	gke_cluster_role_bindings = {
		for role in var.gke_cluster_service_account_roles : role => role
	}
}

resource "tfe_workspace" "workspace" {
	for_each = local.workspace_configs

	name         = each.value.workspace_name
	organization = var.tfc_organization
	description  = "Managed by Terraform for ${each.value.display_name}."
	tag_names    = distinct(concat(var.workspace_tags, [each.key]))
}

resource "google_service_account" "workspace" {
	for_each = local.workspace_configs

	project      = var.gcp_service_project_id
	account_id   = each.value.service_account_id
	display_name = each.value.display_name
	description  = each.value.description
}

resource "google_project_iam_member" "workspace" {
	for_each = local.workspace_role_bindings

	project = var.gcp_service_project_id
	role    = each.value.role
	member  = "serviceAccount:${google_service_account.workspace[each.value.workspace_key].email}"
}

resource "google_service_account" "gke_cluster" {
	project      = var.gcp_service_project_id
	account_id   = var.gke_cluster_service_account_id
	display_name = "GKE cluster service account"
	description  = "Service account intended for the GKE cluster and its node pools."
}

resource "google_service_account_iam_member" "gke_cluster_usage" {
	for_each = local.gke_cluster_service_account_bindings

	service_account_id = google_service_account.gke_cluster.name
	role               = each.value
	member             = "serviceAccount:${google_service_account.workspace["gke"].email}"
}

resource "google_project_iam_member" "gke_cluster" {
	for_each = local.gke_cluster_role_bindings

	project = var.gcp_service_project_id
	role    = each.value
	member  = "serviceAccount:${google_service_account.gke_cluster.email}"
}

resource "tfe_variable" "workspace_impersonation_service_account" {
	for_each = local.workspace_configs

	workspace_id = tfe_workspace.workspace[each.key].id
	key          = "GOOGLE_IMPERSONATE_SERVICE_ACCOUNT"
	value        = google_service_account.workspace[each.key].email
	category     = "env"
	description  = "Google service account email used for provider impersonation."
}

resource "tfe_variable" "workspace_project_id" {
	for_each = local.workspace_configs

	workspace_id = tfe_workspace.workspace[each.key].id
	key          = "TF_VAR_gcp_service_project_id"
	value        = var.gcp_service_project_id
	category     = "env"
	description  = "Service project ID exposed to Terraform runs."
}

resource "tfe_variable" "workspace_region" {
	for_each = local.workspace_configs

	workspace_id = tfe_workspace.workspace[each.key].id
	key          = "TF_VAR_gcp_region"
	value        = var.gcp_region
	category     = "env"
	description  = "Default Google Cloud region exposed to Terraform runs."
}
