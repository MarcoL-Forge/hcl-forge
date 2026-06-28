resource "google_container_node_pool" "pool" {
  name = "primary-pool"

  node_config {
    machine_type = "e2-medium"
  }

  tags = ["legacy-node"]
}
