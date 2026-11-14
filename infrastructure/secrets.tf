# Secret Manager is used (Good practice), but secret version is pinned statically,
# and fallback/sensitive DB connection strings are embedded directly in the secret.
resource "google_secret_manager_secret" "todo_secrets" {
  secret_id = "todo-app-secrets"

  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "todo_secrets_v1" {
  secret = google_secret_manager_secret.todo_secrets.id

  # Insecure: Storing bundled master secrets / admin bypass keys inside one unrotated payload
  secret_data = jsonencode({
    admin_api_key = "todo-master-key-prod-2026"
    jwt_secret    = "weak-todo-jwt-signing-key"
  })
}
