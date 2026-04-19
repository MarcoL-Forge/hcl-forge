gcp_service_project_id = "example-service-project"
gcp_host_project_id    = "example-host-project"
gcp_region             = "us-central1"

gke_workspace_service_account_id             = "tf-gke-workspace"
service_project_workspace_service_account_id = "tf-service-project-workspace"

gke_workspace_roles = [
	"roles/container.admin",
	"roles/compute.networkAdmin",
	"roles/iam.serviceAccountUser",
]

gke_workspace_host_project_roles = [
	"roles/compute.networkUser",
]

service_project_workspace_roles = [
	"roles/resourcemanager.projectIamAdmin",
	"roles/serviceusage.serviceUsageAdmin",
	"roles/compute.networkAdmin",
]