# Container Security on Cloud Run — Workshop Runbook

**90-minute workshop:** Attendees inspect, deploy, break, and harden a Go Todo service on Google Cloud Run across three distinct operational layers:

- **Infrastructure:** IAM permissions, custom Service Account scope, and public vs. private ingress.
- **Container:** Base image bloat, root execution (`USER nonroot`), and CVE management.
- **Application:** Secret injection, debug routes, and network timeouts.

---

> [!CAUTION]
> This repository is **intentionally insecure** for workshop purposes.
> Every flaw is documented in [VULNERABILITIES.md](./VULNERABILITIES.md).

> [!TIP]
> Need to install prerequisites (`gcloud`, Docker, Terraform, Trivy, etc.)?
> Check out the [GETTING_STARTED.md](./GETTING_STARTED.md) guide before beginning.

---

## Repository Layout

```text
.
├── application/             # Go 1.26 + Fiber v3 Todo service
│   ├── main.go              # REST API logic and diagnostics
│   ├── go.mod               # Dependencies (fiber v3.3.0)
│   ├── go.sum
│   ├── Dockerfile           # Container build definition
│   └── README.md            # Local application setup & test guide
├── infrastructure/          # Terraform configurations
│   ├── main.tf              # Provider, APIs, and Artifact Registry
│   ├── service_account.tf   # Custom service account & IAM bindings
│   ├── secrets.tf           # Secret Manager configuration
│   ├── cloud_run.tf         # Cloud Run service definition
│   ├── variables.tf         # Input variables (project_id, region, image)
│   ├── outputs.tf           # Endpoints and IDs
│   └── README.md            # Infrastructure setup & variables guide
├── scripts/
│   ├── deploy_todo.sh       # Automated build & deploy script
│   ├── attack.sh            # Exploit and test script
│   └── cleanup.sh           # Resource teardown script
├── reports/
│   └── SECURITY_REPORT.md   # Generated 3-layer security audit report
├── GETTING_STARTED.md       # Prerequisite tools installation guide
├── VULNERABILITIES.md       # Intentional flaws reference (workshop guide)
└── README.md                # This file (End-to-End Workshop Guide)
```

---

## Step 0: GCP Project Preparation

Before deploying, ensure you have an active Google Cloud project with billing enabled.

### 1. Set Environment Variables
```bash
export PROJECT_ID="devfest-cloud-sby-2026-csec-run"
export REGION="us-central1"
```

### 2. Create the Project
```bash
gcloud projects create "$PROJECT_ID" --name="DevFest Cloud 2026"
```

### 3. Link Billing Account
```bash
# List your billing account IDs
gcloud billing accounts list

# Link your billing account
export BILLING_ACCOUNT_ID="YOUR_BILLING_ACCOUNT_ID"
gcloud billing projects link "$PROJECT_ID" --billing-account="$BILLING_ACCOUNT_ID"
```

### 4. Set Active Project & Configure ADC
```bash
gcloud config set project "$PROJECT_ID"

# If running on local laptop terminal:
gcloud auth application-default login
gcloud auth application-default set-quota-project "$PROJECT_ID"
```

---
---

## Step 1: Deploy to Cloud Run

You can deploy using either the automated script or manual step-by-step commands.

### Option A: Automated Build & Deploy (Recommended)

From the repository root:

```bash
./scripts/deploy_todo.sh
```

This script:
1. Ensures the `todo-repo` Artifact Registry repository exists in `$REGION`.
2. Packages and builds `application/` via **Google Cloud Build** (`us-central1-docker.pkg.dev/$PROJECT_ID/todo-repo/todo-service:stable`).
3. Provisions the infrastructure with Terraform and deploys the container to Cloud Run.
4. Outputs the live service URL.

---
---

### Option B: Manual Step-by-Step Deploy

#### 1. Build & Push the Todo Container Image First
Build the custom container image and push it to Artifact Registry:

* **Choice 1: Using Cloud Build (Fastest, no local Docker needed):**
  ```bash
  gcloud builds submit ./application \
    --project="$PROJECT_ID" \
    --tag="${REGION}-docker.pkg.dev/${PROJECT_ID}/todo-repo/todo-service:stable"
  ```

* **Choice 2: Using Local Docker (`docker build` & `docker push`):**
  ```bash
  # Authenticate Docker CLI to Artifact Registry
  gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet

  # Build targeting linux/amd64 (critical for Apple Silicon/M-series Mac users)
  docker build --platform linux/amd64 \
    -t "${REGION}-docker.pkg.dev/${PROJECT_ID}/todo-repo/todo-service:stable" \
    ./application

  # Push to your GCP Artifact Registry
  docker push "${REGION}-docker.pkg.dev/${PROJECT_ID}/todo-repo/todo-service:stable"
  ```

#### 2. Provision the Infrastructure via Terraform
Now navigate to `infrastructure/` and run `terraform apply`:
```bash
cd infrastructure
terraform init
terraform apply -var="project_id=$PROJECT_ID"
cd ..
```
*Note: In Git, `infrastructure/variables.tf` already points to `todo-service:stable` by default. If you ever need to apply Terraform on a fresh project before the image exists, you can temporarily uncomment the `hello` placeholder in `variables.tf`.*

---
---

## Step 2: Test the Live Service

Extract your live Cloud Run URL:
```bash
SERVICE_URL=$(cd infrastructure && terraform output -raw service_url)
echo "Service URL: $SERVICE_URL"
```

### 1. Healthcheck Probe
```bash
curl -i "$SERVICE_URL/healthz"
```

### 2. Fetch Todo Items (Public Ingress)
```bash
curl -i "$SERVICE_URL/api/v1/todos"
```

### 3. Inspect Leaked Diagnostics (The Flaw)
Because `DEBUG=true` is enabled in the staging deployment, the internal diagnostic endpoint dumps all environment variables, including `APP_CONFIG_SECRETS`:
```bash
curl -s "$SERVICE_URL/api/v1/system/diagnostics" | jq .
```

### 4. Delete an Item with the Leaked Admin Key
Extract the key from the diagnostics endpoint and issue an administrative delete:
```bash
KEY=$(curl -s "$SERVICE_URL/api/v1/system/diagnostics" | jq -r '.environment_variables.TODO_ADMIN_KEY')

curl -i -X DELETE "$SERVICE_URL/api/v1/todos/1" \
  -H "X-Admin-Key: $KEY"
```

---

## Step 3: Security Review with Antigravity Agent

Instead of manually reviewing code and configurations, leverage the custom workspace skill **`container-security-review`** included in this repository.

The skill is defined in [`.agents/skills/container-security-review/`](./.agents/skills/container-security-review/SKILL.md) and instructs the Antigravity agent to evaluate the project across three operational layers (**Platform**, **Container / Runtime**, **Application**) mapped against **OWASP Top 10:2025**.

### 1. Trigger the Skill & Generate the Report
In your Antigravity conversation or CLI prompt, ask the agent to run the review and output a structured report to a file:

```text
Perform a full security review of the application, container, and infrastructure using the container-security-review skill, and generate a comprehensive security report saved to reports/SECURITY_REPORT.md.
```

### 2. What the Agent Inspects

The agent autonomously walks through the 3-layer audit checklist:
- **Layer 3 — Platform:** Cloud Run ingress, unauthenticated invoker permissions (`allUsers`), IAM service account blast radius, and Secret Manager exposure.
- **Layer 2 — Container:** Base image bloat (`golang:1.26` vs. distroless/alpine), lack of `USER` directive (running as root), baked secrets in `ENV` image layers, and PID 1 signal handling.
- **Layer 1 — Application:** Fallback secrets in code, debug diagnostic endpoints dumping environment variables, error handling reflection leaks, and dependency integrity.

### 3. Apply the Hardened Fixes
Once the agent provides findings with file links and severity mappings, you can prompt it to automatically harden your codebase:

```text
Apply the recommended container fixes to application/Dockerfile and harden cloud_run.tf.
```

---

## Step 4: Cleanup

To destroy all provisioned cloud resources:

```bash
cd infrastructure
terraform destroy -var="project_id=$PROJECT_ID"
```

To delete the entire project and guarantee zero ongoing charges:
```bash
gcloud projects delete "$PROJECT_ID"
```
