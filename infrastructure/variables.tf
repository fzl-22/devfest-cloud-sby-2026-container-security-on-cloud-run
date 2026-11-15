variable "project_id" {
  type        = string
  description = "The GCP project ID where resources will be deployed (max 30 chars)"
  default     = "devfestcloud-sby-2026-csec-run"
}

variable "region" {
  type        = string
  description = "GCP Region for Cloud Run and Artifact Registry"
  default     = "us-central1" # Lowest pricing tier for Cloud Run & Artifact Registry + full Free Tier coverage
}

variable "service_name" {
  type        = string
  description = "The Cloud Run service name"
  default     = "todo-app-default"
}

# ==============================================================================
# Container Image Configuration
# ==============================================================================
# Defaults to null; cloud_run.tf coalesces to:
#   ${var.region}-docker.pkg.dev/${var.project_id}/todo-repo/todo-service:stable
# ==============================================================================
variable "image" {
  type        = string
  description = "Container image URI for the Todo application. Leave null to default to <region>-docker.pkg.dev/<project_id>/todo-repo/todo-service:stable"
  default     = null
}
