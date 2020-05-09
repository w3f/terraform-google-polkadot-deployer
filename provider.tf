provider "google" {
  project     = var.gcp_project_id
  version     = "~>2.16"
}

provider "google-beta" {
  project     = var.gcp_project_id
  version     = "~>2.16"
}

provider "template" {
  version     = "~>2.1"
}
