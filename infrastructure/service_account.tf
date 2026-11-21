resource "google_service_account" "todo_sa" {
  project      = var.project_id
  account_id   = "todo-app-sa"
  display_name = "Todo Application Service Account"
}

# ==============================================================================
# Hardened Least-Privilege IAM Configuration
# ==============================================================================

# 1. Datastore/Firestore User for todo persistence
resource "google_project_iam_member" "datastore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.todo_sa.email}"
}

# 2. Scoped Secret Accessor: Strictly bound to the specific secret resource,
# eliminating project-wide secret inspection blast radius (V-PLT-03).
resource "google_secret_manager_secret_iam_member" "todo_secret_accessor" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.todo_secrets.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.todo_sa.email}"
}

# ==============================================================================
# Cloud Build & Artifact Registry Permissions
# ==============================================================================
# Allow the project's default Cloud Build & Compute service accounts to push images to Artifact Registry
data "google_project" "current" {
  project_id = var.project_id
}

# Legacy Cloud Build service account
resource "google_project_iam_member" "cloudbuild_registry_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${data.google_project.current.number}@cloudbuild.gserviceaccount.com"

  depends_on = [google_project_service.cloudbuild]
}

# Default Compute service account used by Cloud Build in modern GCP projects
resource "google_project_iam_member" "compute_cloudbuild_storage" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = "serviceAccount:${data.google_project.current.number}-compute@developer.gserviceaccount.com"

  depends_on = [
    google_project_service.cloudbuild,
    google_project_service.compute,
  ]
}

resource "google_project_iam_member" "compute_cloudbuild_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${data.google_project.current.number}-compute@developer.gserviceaccount.com"

  depends_on = [
    google_project_service.cloudbuild,
    google_project_service.compute,
  ]
}

resource "google_project_iam_member" "compute_cloudbuild_logs" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${data.google_project.current.number}-compute@developer.gserviceaccount.com"

  depends_on = [
    google_project_service.cloudbuild,
    google_project_service.compute,
  ]
}
