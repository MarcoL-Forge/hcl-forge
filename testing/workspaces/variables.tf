variable "tfc_hostname" {
	description = "Terraform Cloud or Terraform Enterprise hostname."
	type        = string
	default     = "app.terraform.io"
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
	default     = ["managed-by-terraform"]
}

variable "gke_workspace_name" {
	description = "Name of the Terraform workspace that manages GKE."
	type        = string
	default     = "gke"
}

variable "service_project_workspace_name" {
	description = "Name of the Terraform workspace that manages the GCP service project."
	type        = string
	default     = "gcp-service-project"
}

variable "gke_workspace_service_account_id" {
	description = "Service account ID used by the Terraform workspace that manages GKE."
	type        = string
	default     = "tf-gke-workspace"
}

variable "service_project_workspace_service_account_id" {
	description = "Service account ID used by the Terraform workspace that manages the GCP service project."
	type        = string
	default     = "tf-service-project-workspace"
}

variable "gke_cluster_service_account_id" {
	description = "Service account ID used by the GKE cluster or its node pools."
	type        = string
	default     = "gke-cluster"
}

variable "gke_workspace_roles" {
	description = "Project-level IAM roles granted to the Terraform workspace that manages GKE."
	type        = list(string)
	default = [
		"roles/container.admin",
		"roles/compute.networkAdmin",
	]
}

variable "gke_cluster_service_account_user_roles" {
	description = "IAM roles granted on the GKE cluster service account to the Terraform workspace that manages GKE."
	type        = list(string)
	default = [
		"roles/iam.serviceAccountUser",
	]
}

variable "service_project_workspace_roles" {
	description = "Project-level IAM roles granted to the Terraform workspace that manages the GCP service project."
	type        = list(string)
	default = [
		"roles/resourcemanager.projectIamAdmin",
		"roles/serviceusage.serviceUsageAdmin",
		"roles/compute.networkAdmin",
	]
}

variable "gke_cluster_service_account_roles" {
	description = "Project-level IAM roles granted to the GKE cluster service account."
	type        = list(string)
	default = [
		"roles/logging.logWriter",
		"roles/monitoring.metricWriter",
		"roles/monitoring.viewer",
		"roles/stackdriver.resourceMetadata.writer",
		"roles/artifactregistry.reader",
	]
}