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
# The default in Git points directly to the real application image.
#
# NOTE FOR FRESH PROJECT BOOTSTRAP:
# If you are deploying to a brand-new GCP project before building the image,
# temporarily uncomment the 'hello' placeholder below for your first apply,
# OR build and push the container first with `gcloud builds submit`.
# ==============================================================================
variable "image" {
  type        = string
  description = "Container image URI for the Todo application. Leave null to default to <region>-docker.pkg.dev/<project_id>/todo-repo/todo-service:stable"
  default     = null
}
