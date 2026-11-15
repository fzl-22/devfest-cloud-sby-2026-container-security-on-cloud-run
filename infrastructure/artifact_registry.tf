# Artifact Registry for Todo container images
resource "google_artifact_registry_repository" "todo_repo" {
  location      = var.region
  repository_id = "todo-repo"
  description   = "Docker repository for Todo API"
  format        = "DOCKER"

  depends_on = [google_project_service.artifactregistry]
}
