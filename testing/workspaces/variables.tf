variable "tfc_hostname" {
	description = "Terraform Cloud or Terraform Enterprise hostname."
	type        = string
}

variable "tfc_organization" {
	description = "Terraform Cloud or Terraform Enterprise organization name."
	type        = string
}

variable "gcp_service_project_id" {
	description = "Google Cloud service project where the service accounts will be created."
	type        = string
}

variable "gcp_region" {
	description = "Default Google Cloud region for the provider and Terraform workspaces."
	type        = string
}

variable "workspace_tags" {
	description = "Tags applied to both Terraform workspaces."
	type        = list(string)
}

variable "gke_workspace_name" {
	description = "Name of the Terraform workspace that manages GKE."
	type        = string
}

variable "service_project_workspace_name" {
	description = "Name of the Terraform workspace that manages the GCP service project."
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

variable "gke_cluster_service_account_id" {
	description = "Service account ID used by the GKE cluster or its node pools."
	type        = string
}

variable "gke_workspace_roles" {
	description = "Project-level IAM roles granted to the Terraform workspace that manages GKE."
	type        = list(string)
}

variable "gke_cluster_service_account_user_roles" {
	description = "IAM roles granted on the GKE cluster service account to the Terraform workspace that manages GKE."
	type        = list(string)
}

variable "service_project_workspace_roles" {
	description = "Project-level IAM roles granted to the Terraform workspace that manages the GCP service project."
	type        = list(string)
}

variable "gke_cluster_service_account_roles" {
	description = "Project-level IAM roles granted to the GKE cluster service account."
	type        = list(string)
}