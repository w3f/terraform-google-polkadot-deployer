terraform {
  backend "gcs" {
    bucket  = "pd-tf-state-${var.deployment_name}"
    prefix  = "terraform/state/${var.cluster_name}"
  }
}
