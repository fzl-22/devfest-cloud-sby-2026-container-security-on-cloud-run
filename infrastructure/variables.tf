variable "project_id" {
  type        = string
  description = "The GCP project ID where resources will be deployed"
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

variable "image" {
  type        = string
  description = "Container image URI for the Todo application"
  default     = "us-docker.pkg.dev/cloudrun/container/hello" # Placeholder sample container until custom todo image is built
}
