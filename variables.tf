variable "cluster_name" {
  description = "Name of the GKE cluster"
  default     = "polkadot-deployer"
  type        = string
}

variable "deployment_name" {
  description = "Name of this polkadot deployment"
  default     = "polkadot"
  type        = string
}

variable "location" {
  description = "GKE location"
  type        = string
}

variable "node_count" {
  description = "Size of GKE cluster"
  default     = 2
  type        = number
}

variable "machine_type" {
  description = "The name of a GKE machine type"
  default     = "n1-standard-2"
  type        = string
}

variable "k8s_version" {
  description = "Kubernetes version to use for the cluster"
  default     = "1.15.11-gke.11"
  type        = string
}

variable "image_type" {
  description = "The image type to use for the cluster nodes"
  default     = "UBUNTU"
  type        = string
}

variable "gcloud_path" {
  description = "Path to gcloud binary"
  default     = "/usr/bin/gcloud"
  type        = string
}

variable "gcp_project_id" {
  description = "Google cloud project id used for terraform state"
  type        = string
}

variable "gcp_credentials" {
  description = "Either the path to or the contents of a GCP service account key file in JSON format"
  type        = string
}
