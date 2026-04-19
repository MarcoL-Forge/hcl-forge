variable "gcp_service_project_id" {
	description = "Google Cloud service project where the workspace service accounts will be created."
	type        = string
}

variable "gcp_host_project_id" {
	description = "Google Cloud host project that contains shared VPC networks and subnets used by GKE."
	type        = string
}

variable "gcp_region" {
	description = "Default Google Cloud region for the provider."
	type        = string
}

variable "gke_workspace_service_account_id" {
	description = "Service account ID used by the Terraform workspace that manages GKE."
	type        = string
}

variable "service_project_workspace_service_account_id" {
	description = "Service account ID used by the Terraform workspace that manages the GCP service project."
	type        = string
}

variable "gke_workspace_roles" {
	description = "Service-project IAM roles granted to the Terraform workspace service account for GKE management."
	type        = list(string)
}

variable "gke_workspace_host_project_roles" {
	description = "Host-project IAM roles granted to the Terraform workspace service account for shared VPC network access required by GKE."
	type        = list(string)
}

variable "service_project_workspace_roles" {
	description = "Project-level IAM roles granted to the Terraform workspace service account for service project management."
	type        = list(string)
}
