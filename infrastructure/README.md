# Todo Application — Infrastructure (Realistic but Insecure)

This directory provisions a **realistic, believable Cloud Run setup** for the Todo application using Terraform. 

At first glance, it looks like a clean production configuration:
- Dedicated custom Service Account (`todo-app-sa`).
- Secret Manager integration (`secret_key_ref`).
- Resource limits and startup probes defined.
- Managed Artifact Registry repository.

However, like many real-world services rushed to staging or production, it contains **architectural flaws and privilege escalations** that Gemini and vulnerability scanners can uncover.

---

## The Subtle Flaws in This Architecture

| Category | What It Looks Like | The Real Security Flaw |
|---|---|---|
| **IAM Scope** | Uses a dedicated service account (`todo-app-sa`) | Granted project-wide `roles/storage.admin` and `roles/secretmanager.secretAccessor` instead of resource-specific least-privilege. |
| **Secret Management** | Mounts secrets via Secret Manager | Injects an entire multi-key JSON blob into a single environment variable (`APP_CONFIG_SECRETS`), which is easily leaked by process dumps or crash logs. |
| **Ingress Control** | Public REST API (`allUsers`) | No Cloud Armor / API Gateway or IAM authorization layer in front of administrative routes. |
| **Operational Flags** | Configured environment variables | `DEBUG=true` left on in the deployment template. |

---

## Prerequisites

1. **Google Cloud SDK (`gcloud`)** installed and authenticated:
   ```bash
   gcloud auth login
   gcloud auth application-default login
   ```
2. **Terraform** (`>= 1.5.0`) installed.
3. An active GCP project with **billing enabled**.

---

## Step-by-Step Deployment Guide

### Step 1: Set Your Environment Variables

```bash
export PROJECT_ID="your-google-cloud-project-id"
export REGION="us-central1" # Cheapest tier (Tier 1) with full GCP Free Tier coverage
```

Ensure your `gcloud` CLI is pointing to the project:
```bash
gcloud config set project "$PROJECT_ID"
```

---

### Step 2: Initialize Terraform

Navigate to the `infrastructure` directory and initialize provider plugins:

```bash
cd infrastructure
terraform init
```

---

### Step 3: Configure Project Variables

Copy the example variables file:
```bash
cp terraform.tfvars.example terraform.tfvars
```

Update `terraform.tfvars`:
```hcl
project_id = "your-google-cloud-project-id"
region     = "us-central1"
```

---

### Step 4: Plan the Deployment

Run Terraform plan to inspect the resources:

```bash
terraform plan -var="project_id=$PROJECT_ID"
```

Resources to be created:
* APIs: `run.googleapis.com`, `artifactregistry.googleapis.com`, `secretmanager.googleapis.com`
* Artifact Registry: `todo-repo`
* Service Account: `todo-app-sa` with project IAM bindings
* Secret Manager: `todo-app-secrets` (JSON payload)
* Cloud Run: `todo-app-default` with probe and secret reference (max instances: 2)

---

### Step 5: Provision the Infrastructure

Apply the configuration and confirm when prompted with `yes`:

```bash
terraform apply -var="project_id=$PROJECT_ID"
```

Provisioning takes approximately 1 to 2 minutes.

---

### Step 6: Inspect the Deployment

View the outputs:

```bash
terraform output
```

You will see:
* `service_url`: The live Cloud Run URL.
* `service_account_email`: `todo-app-sa@<project-id>.iam.gserviceaccount.com`.
* `secret_id`: `todo-app-secrets`.

Test the live endpoint:
```bash
SERVICE_URL=$(terraform output -raw service_url)
curl -i "$SERVICE_URL"
```

---

## Why This Serves the Workshop Better

When attendees feed these Terraform files to Gemini in the review phase:
1. Gemini won't give trivial advice like *"create a service account"*, because one already exists.
2. Instead, it must evaluate **least privilege**:
   - Why does a Todo app need `storage.admin` across the entire project?
   - Why is `secretmanager.secretAccessor` granted at the project level rather than scoped to `google_secret_manager_secret.todo_secrets.id`?
   - Why are bundled secrets dumped into an environment variable instead of individual secret bindings?

---

## Teardown / Cleanup

To delete all provisioned resources:

```bash
terraform destroy -var="project_id=$PROJECT_ID"
```
