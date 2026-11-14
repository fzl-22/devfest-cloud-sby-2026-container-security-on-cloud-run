# ==============================================================================
# REALISTIC PRODUCTION SETUP — WITH ARCHITECTURAL FLAWS
# ==============================================================================
# Looks like a professional setup:
# - Custom Service Account (todo_sa)
# - Secret Manager integration via secret_key_ref
# - Healthcheck probes configured
# - Scaling limits declared
#
# But with real-world subtle flaws:
# 1. Ingress allows allUsers without Cloud Run IAM token validation (A01).
# 2. Service account has project-wide roles (storage.admin, secretmanager at project level).
# 3. Debug flags left enabled in environment variables.
# 4. No VPC Egress connector / unrestricted outbound internet.
# ==============================================================================

resource "google_cloud_run_v2_service" "todo_service" {
  name                = var.service_name
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    # 1. Custom Service Account is used (looks good!)
    service_account = google_service_account.todo_sa.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image = var.image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }

      env {
        name  = "DEBUG"
        value = "true"
      }

      env {
        name  = "ENVIRONMENT"
        value = "staging"
      }

      env {
        name = "APP_CONFIG_SECRETS"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.todo_secrets.secret_id
            version = "latest"
          }
        }
      }

      # Standard startup probe (looks like real enterprise config)
      startup_probe {
        initial_delay_seconds = 2
        timeout_seconds       = 3
        period_seconds        = 5
        failure_threshold     = 3
        tcp_socket {
          port = 8080
        }
      }
    }
  }

  depends_on = [
    google_project_service.run,
    google_secret_manager_secret_version.todo_secrets_v1,
    google_project_iam_member.secret_broad
  ]
}

# Public ingress: Frontend web/mobile apps hit this directly without Google IAM
resource "google_cloud_run_v2_service_iam_member" "public_ingress" {
  name     = google_cloud_run_v2_service.todo_service.name
  location = google_cloud_run_v2_service.todo_service.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
