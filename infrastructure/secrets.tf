# Secret Manager is used (Good practice), but all service secrets are bundled into
# a single JSON blob mounted as an environment variable (Anti-pattern: excessive blast radius).
resource "google_secret_manager_secret" "todo_secrets" {
  secret_id = "todo-app-secrets"

  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "todo_secrets_v1" {
  secret = google_secret_manager_secret.todo_secrets.id

  secret_data = jsonencode({
    # Database connection pool credentials
    db_host     = "10.128.0.5"
    db_name     = "todos_production"
    db_user     = "todo_app_user"
    db_password = "pG_s3cur3_p@ssw0rd_2026!"

    # Core authentication & administration
    admin_api_key   = "todo-master-key-prod-2026"
    jwt_signing_key = "super-secret-jwt-signing-key-devfest-2026"

    # External SaaS integrations
    sentry_dsn       = "https://pubkey@o123456.ingest.sentry.io/78910"
    sendgrid_api_key = "SG.v2_enterprise_mail_mock_token_77a9"
  })
}
