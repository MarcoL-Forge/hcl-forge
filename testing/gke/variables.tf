variable "gcp_service_project_id" {
	description = "Google Cloud service project where the GKE cluster will be created."
	type        = string
}

variable "gcp_host_project_id" {
	description = "Google Cloud host project that owns the shared VPC network and subnet used by GKE."
	type        = string
}

variable "gcp_region" {
	description = "Region where the GKE cluster and node pool will be created."
	type        = string
}

variable "environment" {
	description = "Environment prefix added to the cluster name."
	type        = string
	default     = "dev"
}

variable "cluster_name" {
	description = "Base name for the GKE cluster."
	type        = string
	default     = "main"
}

variable "network_name" {
	description = "Shared VPC network name in the host project."
	type        = string
}

variable "subnetwork_name" {
	description = "Shared VPC subnetwork name in the host project."
	type        = string
}

variable "pods_secondary_range_name" {
	description = "Secondary IP range name used for GKE pods."
	type        = string
}

variable "services_secondary_range_name" {
	description = "Secondary IP range name used for GKE services."
	type        = string
}

variable "gke_node_service_account_email" {
	description = "Service account email used by GKE nodes."
	type        = string
}

variable "deletion_protection" {
	description = "Whether Terraform should enable deletion protection on the cluster."
	type        = bool
	default     = false
}

variable "enable_private_nodes" {
	description = "Whether GKE nodes should use private IP addresses only."
	type        = bool
	default     = true
}

variable "enable_private_endpoint" {
	description = "Whether the GKE control plane endpoint should be private only."
	type        = bool
	default     = false
}

variable "master_ipv4_cidr_block" {
	description = "CIDR block used by the GKE control plane."
	type        = string
	default     = "172.16.0.0/28"
}

variable "release_channel" {
	description = "GKE release channel for the cluster."
	type        = string
	default     = "REGULAR"
}

variable "maintenance_recurrence" {
	description = "RFC5545 recurrence for the maintenance window."
	type        = string
	default     = "FREQ=WEEKLY;BYDAY=SA,SU"
}

variable "maintenance_start_time" {
	description = "Maintenance window start time in RFC3339 format."
	type        = string
	default     = "2026-01-03T02:00:00Z"
}

variable "maintenance_end_time" {
	description = "Maintenance window end time in RFC3339 format."
	type        = string
	default     = "2026-01-03T06:00:00Z"
}

variable "node_machine_type" {
	description = "Machine type used by the default node pool."
	type        = string
	default     = "e2-standard-4"
}

variable "node_disk_size_gb" {
	description = "Disk size for nodes in the default node pool."
	type        = number
	default     = 100
}

variable "node_disk_type" {
	description = "Disk type for nodes in the default node pool."
	type        = string
	default     = "pd-balanced"
}

variable "node_image_type" {
	description = "Image type for nodes in the default node pool."
	type        = string
	default     = "COS_CONTAINERD"
}

variable "use_spot_nodes" {
	description = "Whether to use spot nodes in the default node pool."
	type        = bool
	default     = false
}

variable "min_node_count" {
	description = "Minimum number of nodes in the default node pool."
	type        = number
	default     = 1
}

variable "max_node_count" {
	description = "Maximum number of nodes in the default node pool."
	type        = number
	default     = 3
}

variable "cluster_labels" {
	description = "Labels applied to the GKE cluster."
	type        = map(string)
	default = {
		managed_by = "terraform"
	}
}

variable "node_labels" {
	description = "Labels applied to nodes in the default node pool."
	type        = map(string)
	default = {
		workload = "general"
	}
}

variable "node_tags" {
	description = "Network tags applied to nodes in the default node pool."
	type        = list(string)
	default     = ["gke"]
}
