output "terraform_workspace_names" {
	description = "Terraform workspaces created by this configuration."
	value = {
		for workspace_key, workspace in tfe_workspace.workspace : workspace_key => workspace.name
	}
}

output "terraform_workspace_service_accounts" {
	description = "Service account emails mapped to the Terraform workspaces that use them."
	value = {
		for workspace_key, service_account in google_service_account.workspace : workspace_key => service_account.email
	}
}

output "gke_cluster_service_account_email" {
	description = "Service account email created for the GKE cluster."
	value       = google_service_account.gke_cluster.email
}
