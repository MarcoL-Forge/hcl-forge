output "cluster_name" {
	description = "Name of the GKE cluster."
	value       = google_container_cluster.this.name
}

output "cluster_location" {
	description = "Region where the GKE cluster was created."
	value       = google_container_cluster.this.location
}

output "cluster_endpoint" {
	description = "GKE control plane endpoint."
	value       = google_container_cluster.this.endpoint
}

output "cluster_ca_certificate" {
	description = "Base64-encoded public CA certificate for the GKE cluster."
	value       = google_container_cluster.this.master_auth[0].cluster_ca_certificate
	sensitive   = true
}

output "node_pool_name" {
	description = "Name of the default node pool."
	value       = google_container_node_pool.default.name
}
