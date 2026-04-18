output "workspace_service_account_emails" {
	description = "Service account emails created for the Terraform workspaces."
	value = {
		for workspace_key, service_account in google_service_account.workspace : workspace_key => service_account.email
	}
}

output "workspace_service_account_ids" {
	description = "Fully qualified service account IDs created for the Terraform workspaces."
	value = {
		for workspace_key, service_account in google_service_account.workspace : workspace_key => service_account.name
	}
}
