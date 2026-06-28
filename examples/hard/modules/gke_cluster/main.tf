resource "google_container_node_pool" "primary" {
  name = "primary-pool"

  node_config {
    machine_type = "e2-medium"
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}
