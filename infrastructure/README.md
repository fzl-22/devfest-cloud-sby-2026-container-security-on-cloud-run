# Todo Application — Infrastructure (Terraform)

This directory contains the Terraform configuration for provisioning the Cloud Run service, custom Service Account, IAM bindings, and Secret Manager secrets for the Todo microservice.

---

## Architecture Components

* **`main.tf`**: Terraform version requirements and Google provider configuration.
* **`services.tf`**: Enables required Google Cloud APIs (`run`, `artifactregistry`, `secretmanager`, `cloudbuild`).
* **`artifact_registry.tf`**: Provisions the Artifact Registry repository (`todo-repo`) for container images.
* **`service_account.tf`**: Provisions the dedicated `todo-app-sa` service account with broad permissions (`storage.admin`, `secretmanager.secretAccessor`, `datastore.user`).
* **`secrets.tf`**: Provisions the Secret Manager secret (`todo-app-secrets`) containing bundled JSON configuration (database credentials, auth tokens, admin key, SaaS API tokens).
* **`cloud_run.tf`**: Deploys the Cloud Run v2 service (`todo-app-default`), scaling boundaries (`max_instance_count = 2`), TCP startup probes, public ingress (`allUsers`), and secret mounts.
* **`variables.tf`**: Input variables for project ID, region (default `us-central1`), image URI, and service name.
* **`outputs.tf`**: Exposes the live Cloud Run endpoint, service account email, and repository URIs.

---

## Infrastructure Setup & Prerequisites

### 1. GCP Project & Authentication
Ensure you have authenticated with your GCP project:
```bash
export PROJECT_ID="devfest-cloud-sby-2026-csec-run"
export REGION="us-central1"

gcloud config set project "$PROJECT_ID"
```

If running locally on your laptop:
```bash
gcloud auth application-default login
gcloud auth application-default set-quota-project "$PROJECT_ID"
```

### 2. Configure Variables
Copy the example variables file:
```bash
cp terraform.tfvars.example terraform.tfvars
```

Update `terraform.tfvars`:
```hcl
project_id = "devfest-cloud-sby-2026-csec-run"
region     = "us-central1"
```

### 3. Initialize Terraform
```bash
terraform init
```

### 4. Plan Infrastructure
```bash
terraform plan -var="project_id=$PROJECT_ID"
```

---

## Outputs & Teardown

To inspect outputs after deployment:
```bash
terraform output
```

To destroy provisioned infrastructure:
```bash
terraform destroy -var="project_id=$PROJECT_ID"
```
