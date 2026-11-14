resource "google_service_account" "todo_sa" {
  account_id   = "todo-app-sa"
  display_name = "Todo Application Service Account"
}

# The Realistic Anti-Pattern:
# To avoid permission issues during staging/MVP development, the team assigned
# broad roles instead of fine-grained permissions.

# 1. Broad Storage Admin instead of specific bucket read/write
resource "google_project_iam_member" "storage_broad" {
  project = var.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.todo_sa.email}"
}

# 2. Broad Datastore/Firestore User for todo persistence
resource "google_project_iam_member" "datastore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.todo_sa.email}"
}

# 3. Secret Manager Admin (or Secret Accessor at PROJECT level instead of single secret)
resource "google_project_iam_member" "secret_broad" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.todo_sa.email}"
}
