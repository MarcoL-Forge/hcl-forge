gcp_service_project_id       = "example-service-project"
gcp_host_project_id          = "example-host-project"
gcp_region                   = "us-central1"
environment                  = "dev"
cluster_name                 = "main"
network_name                 = "shared-vpc"
subnetwork_name              = "gke-subnet"
pods_secondary_range_name    = "gke-pods"
services_secondary_range_name = "gke-services"
gke_node_service_account_email = "gke-node@example-service-project.iam.gserviceaccount.com"
deletion_protection          = false
enable_private_nodes         = true
enable_private_endpoint      = false
master_ipv4_cidr_block       = "172.16.0.0/28"
release_channel              = "REGULAR"
maintenance_recurrence       = "FREQ=WEEKLY;BYDAY=SA,SU"
maintenance_start_time       = "2026-01-03T02:00:00Z"
maintenance_end_time         = "2026-01-03T06:00:00Z"
node_machine_type            = "e2-standard-4"
node_disk_size_gb            = 100
node_disk_type               = "pd-balanced"
node_image_type              = "COS_CONTAINERD"
use_spot_nodes               = false
min_node_count               = 1
max_node_count               = 3

cluster_labels = {
	managed_by = "terraform"
}

node_labels = {
	workload = "general"
}

node_tags = ["gke"]