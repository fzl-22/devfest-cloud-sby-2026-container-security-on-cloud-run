output "service_url" {
  description = "The public endpoint URL of the Todo Cloud Run service"
  value       = google_cloud_run_v2_service.todo_service.uri
}

output "service_account_email" {
  description = "The custom service account email used by the application"
  value       = google_service_account.todo_sa.email
}

output "artifact_registry_repo" {
  description = "Artifact Registry Docker repository URI for pushing the Todo container image"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.todo_repo.repository_id}"
}

output "secret_id" {
  description = "Secret Manager secret ID containing application credentials"
  value       = google_secret_manager_secret.todo_secrets.secret_id
}
