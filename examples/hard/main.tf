module "gke_cluster" {
  source     = "./modules/gke_cluster"
  project_id = "legacy-project"
  node_count = 1
}
