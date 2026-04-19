tfc_hostname         = "app.terraform.io"
tfc_organization     = "example-org"
gcp_service_project_id = "example-service-project"
gcp_region           = "us-central1"

workspace_tags = ["managed-by-terraform"]

gke_workspace_name                      = "gke"
service_project_workspace_name          = "gcp-service-project"
gke_workspace_service_account_id        = "tf-gke-workspace"
service_project_workspace_service_account_id = "tf-service-project-workspace"
gke_cluster_service_account_id          = "gke-cluster"

gke_workspace_roles = [
	"roles/container.admin",
	"roles/compute.networkAdmin",
]

gke_cluster_service_account_user_roles = [
	"roles/iam.serviceAccountUser",
]

service_project_workspace_roles = [
	"roles/resourcemanager.projectIamAdmin",
	"roles/serviceusage.serviceUsageAdmin",
	"roles/compute.networkAdmin",
]

gke_cluster_service_account_roles = [
	"roles/logging.logWriter",
	"roles/monitoring.metricWriter",
	"roles/monitoring.viewer",
	"roles/stackdriver.resourceMetadata.writer",
	"roles/artifactregistry.reader",
]