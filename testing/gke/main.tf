locals {
	full_cluster_name = "${var.environment}-${var.cluster_name}"
	network_self_link = "projects/${var.gcp_host_project_id}/global/networks/${var.network_name}"
	subnetwork_self_link = "projects/${var.gcp_host_project_id}/regions/${var.gcp_region}/subnetworks/${var.subnetwork_name}"
}

resource "google_container_cluster" "this" {
	name     = local.full_cluster_name
	project  = var.gcp_service_project_id
	location = var.gcp_region

	network    = local.network_self_link
	subnetwork = local.subnetwork_self_link

	deletion_protection = var.deletion_protection
	remove_default_node_pool = true
	initial_node_count       = 1

	networking_mode = "VPC_NATIVE"

	ip_allocation_policy {
		cluster_secondary_range_name  = var.pods_secondary_range_name
		services_secondary_range_name = var.services_secondary_range_name
	}

	private_cluster_config {
		enable_private_nodes    = var.enable_private_nodes
		enable_private_endpoint = var.enable_private_endpoint
		master_ipv4_cidr_block  = var.master_ipv4_cidr_block
	}

	release_channel {
		channel = var.release_channel
	}

	workload_identity_config {
		workload_pool = "${var.gcp_service_project_id}.svc.id.goog"
	}

	maintenance_policy {
		recurring_window {
			recurrence = var.maintenance_recurrence

			window {
				start_time = var.maintenance_start_time
				end_time   = var.maintenance_end_time
			}
		}
	}

	resource_labels = var.cluster_labels
}

resource "google_container_node_pool" "default" {
	name     = "${local.full_cluster_name}-default"
	project  = var.gcp_service_project_id
	cluster  = google_container_cluster.this.name
	location = var.gcp_region

	node_count = var.min_node_count

	autoscaling {
		min_node_count = var.min_node_count
		max_node_count = var.max_node_count
	}

	management {
		auto_repair  = true
		auto_upgrade = true
	}

	node_config {
		machine_type    = var.node_machine_type
		service_account = var.gke_node_service_account_email
		disk_size_gb    = var.node_disk_size_gb
		disk_type       = var.node_disk_type
		image_type      = var.node_image_type
		preemptible     = var.use_spot_nodes
		spot            = var.use_spot_nodes

		oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

		labels = var.node_labels
		tags   = var.node_tags
	}

	depends_on = [google_container_cluster.this]
}
