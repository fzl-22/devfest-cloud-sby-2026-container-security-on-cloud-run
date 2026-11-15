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

## Workshop Agenda & Pacing (90 Minutes)

| Timeline | Phase | Workshop Section | Hands-on Objectives |
|:---:|:---|:---|:---|
| **00 – 05m** | **Presentation & Intro** | Slide Deck (PowerPoint) | Welcome, speaker intro, workshop goals, and high-level 3-layer security model overview. |
| **05 – 15m** | **Software Installation & GCP Project Preparation** | [GETTING_STARTED.md](./GETTING_STARTED.md)<br>[Step 0: GCP Project Preparation](#step-0-gcp-project-preparation) | Verify/install prerequisite tools (`gcloud`, Docker, Terraform, Trivy), authenticate `gcloud`, link billing, configure ADC, and set active project. |
| **15 – 30m** | **Deploy & Exploit** | [Step 1: Deploy to Cloud Run](#step-1-deploy-to-cloud-run)<br>[Step 2: Test the Live Service](#step-2-test-the-live-service) | Bootstrap Artifact Registry, build/push container, apply Terraform, and exploit leaked secrets via `/diagnostics`. |
| **30 – 55m** | **AI Audit & Pair Hardening** | [Step 3: Security Review with Antigravity](#step-3-security-review-with-antigravity-agent) | Run `container-security-review` skill; interactively harden Application, Container (distroless + non-root), and Platform layers. |
| **55 – 75m** | **Deterministic Verification** | [Step 4: Deterministic Verification](#step-4-beyond-ai-review--deterministic-measurement--verification) | Run `govulncheck` (AST reachability) and `trivy` (CVE scanning); apply fix-and-verify dependency remediation. |
| **75 – 85m** | **Shift-Left Guardrails** | [Step 5: Shift-Left Security](#step-5-shift-left-security--organization-level-guardrails) | Inspect automated GitHub Actions PR gates, PR summary/artifact/SARIF reporting, tool spectrum, and GCP Org Policies. |
| **85 – 90m** | **Wrap-up & Teardown** | [Step 6: Cleanup](#step-6-cleanup) | Clean up resources via `terraform destroy` or full GCP project deletion, followed by open Q&A. |

---

## Repository Layout

```text
.
├── application/             # Go 1.26 + Fiber v3 Todo service
│   ├── main.go              # Service entrypoint & HTTP server
│   ├── internal/            # Decoupled packages (config, handler, model, store)
│   ├── go.mod / go.sum      # Dependencies (fiber v3.3.0)
│   ├── Dockerfile           # Container build definition
│   └── README.md            # Local application setup & test guide
├── infrastructure/          # Terraform configurations
│   ├── main.tf              # Terraform requirements & Google provider
│   ├── services.tf          # Required Google Cloud APIs
│   ├── artifact_registry.tf # Artifact Registry repository
│   ├── service_account.tf   # Custom service account & IAM bindings
│   ├── secrets.tf           # Secret Manager configuration
│   ├── cloud_run.tf         # Cloud Run service definition
│   ├── variables.tf         # Input variables (project_id, region, image)
│   ├── outputs.tf           # Endpoints and IDs
│   └── README.md            # Infrastructure setup & variables guide
├── reports/
│   ├── SECURITY_ASSESSMENT_REPORT.md   # AI-driven 3-layer security audit report
│   ├── APPLICATION_SCAN_REPORT.md      # Deterministic Go application vulnerability report
│   └── CONTAINER_SCAN_REPORT.md        # Deterministic Trivy container vulnerability report
├── GETTING_STARTED.md       # Prerequisite tools installation guide
├── VULNERABILITIES.md       # Intentional flaws reference (workshop guide)
└── README.md                # This file (End-to-End Workshop Guide)
```

---

## Step 0: GCP Project Preparation

Before deploying, ensure you have an active Google Cloud project with billing enabled.

### 1. Set Environment Variables
```bash
export PROJECT_ID="devfestcloud-sby-2026-csec-run"
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

Deploying to a fresh GCP project follows a 3-stage sequence to ensure resources are created in proper order without dependency deadlocks (Artifact Registry must exist before images can be pushed, and the container image must exist before Cloud Run can deploy it):

### 1. Provision Artifact Registry, APIs & Cloud Build Permissions
First, initialize Terraform and provision Artifact Registry, required APIs, and Cloud Build permissions for the compute service account:
```bash
cd infrastructure
terraform init
terraform apply \
  -target=google_project_service.artifactregistry \
  -target=google_project_service.cloudbuild \
  -target=google_artifact_registry_repository.todo_repo \
  -target=google_project_iam_member.compute_cloudbuild_storage \
  -target=google_project_iam_member.compute_cloudbuild_writer \
  -target=google_project_iam_member.compute_cloudbuild_logs \
  -var="project_id=$PROJECT_ID" \
  -var="region=$REGION"
cd ..
```

### 2. Build & Push the Todo Container Image
With `todo-repo` and Cloud Build permissions provisioned, build and push the container image to Artifact Registry:

* **Choice A: Using Cloud Build (Fastest, no local Docker needed):**
  ```bash
  gcloud builds submit ./application \
    --project="$PROJECT_ID" \
    --tag="${REGION}-docker.pkg.dev/${PROJECT_ID}/todo-repo/todo-service:stable"
  ```

* **Choice B: Using Local Docker (`docker build` & `docker push`):**
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

### 3. Deploy Remaining Infrastructure (Cloud Run, IAM, Secrets)
Now that the container image exists in Artifact Registry, provision Cloud Run and all supporting services:
```bash
cd infrastructure
terraform apply \
  -var="project_id=$PROJECT_ID" \
  -var="region=$REGION"
cd ..
```

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

# Inspect the bundled secret JSON (exposes DB credentials, JWT signing key, admin API key, SaaS tokens)
curl -s "$SERVICE_URL/api/v1/system/diagnostics" | jq -r '.environment_variables.APP_CONFIG_SECRETS | fromjson'
```

### 4. Delete an Item with the Leaked Admin Key
Extract the admin API key from the bundled secrets and issue an administrative delete:
```bash
KEY=$(curl -s "$SERVICE_URL/api/v1/system/diagnostics" | jq -r '.environment_variables.APP_CONFIG_SECRETS | fromjson | .admin_api_key')

curl -i -X DELETE "$SERVICE_URL/api/v1/todos/1" \
  -H "X-Admin-Key: $KEY"
```

---

## Step 3: Security Review with Antigravity Agent

Instead of manually reviewing code and configurations, leverage the custom workspace skill **`container-security-review`** included in this repository.

The skill is defined in [`.agents/skills/container-security-review/`](./.agents/skills/container-security-review/SKILL.md) and instructs the Antigravity agent to evaluate the project across three operational layers (**Platform**, **Container / Runtime**, **Application**) mapped against **OWASP Top 10:2025**.

```mermaid
flowchart TB
    subgraph PLATFORM["Layer 3: Platform (Cloud Infrastructure)"]
        direction TB

        subgraph CONTAINER["Layer 2: Container / Runtime"]
            direction TB

            subgraph APP_LAYER["Layer 1: Application"]
                direction TB
                APP["Application Code & Logic"]

                APP -->|"depends on"| DEPA["Dependency A (Web Framework)"]
                APP -->|"depends on"| DEPB["Dependency B (Database Driver)"]
                APP -->|"depends on"| DEPC["Dependency C (Logging / Utilities)"]
            end

            subgraph OS_ENV["Base Image & System Environment"]
                direction LR
                BASE["Base OS (Debian / Alpine / Distroless)"]
                LIBC["C Standard Library (glibc / musl)"]
                PKGS["Installed System Packages"]
                BASE ~~~ LIBC ~~~ PKGS
            end

            APP_LAYER -->|"runs on"| OS_ENV
            DEPA ~~~ BASE
            DEPB ~~~ LIBC
            DEPC ~~~ PKGS
        end

        subgraph P_CONTROLS["Platform Services & Guardrails"]
            direction LR
            IAM["IAM & Service Accounts"]
            NET["Networking (Ingress / Egress & VPC)"]
            SERVICES["Cloud Services & Secret Store"]
            IAM ~~~ NET ~~~ SERVICES
        end

        CONTAINER -->|"deployed on"| P_CONTROLS
    end

    style PLATFORM fill:rgba(99,102,241,0.06),stroke:#6366f1,stroke-width:2px
    style CONTAINER fill:rgba(14,165,233,0.07),stroke:#0ea5e9,stroke-width:2px
    style APP_LAYER fill:rgba(16,185,129,0.08),stroke:#10b981,stroke-width:2px
    style OS_ENV fill:rgba(245,158,11,0.06),stroke:#f59e0b,stroke-width:1.5px,stroke-dasharray: 4 4
    style P_CONTROLS fill:rgba(168,85,247,0.06),stroke:#a855f7,stroke-width:1.5px,stroke-dasharray: 4 4

    style APP fill:rgba(16,185,129,0.18),stroke:#10b981
    style DEPA fill:rgba(16,185,129,0.12),stroke:#10b981
    style DEPB fill:rgba(16,185,129,0.12),stroke:#10b981
    style DEPC fill:rgba(16,185,129,0.12),stroke:#10b981

    style BASE fill:rgba(245,158,11,0.12),stroke:#f59e0b
    style LIBC fill:rgba(245,158,11,0.12),stroke:#f59e0b
    style PKGS fill:rgba(245,158,11,0.12),stroke:#f59e0b

    style IAM fill:rgba(168,85,247,0.12),stroke:#a855f7
    style NET fill:rgba(168,85,247,0.12),stroke:#a855f7
    style SERVICES fill:rgba(168,85,247,0.12),stroke:#a855f7
```

### 1. Trigger the Skill & Generate the Report
In your Antigravity conversation or CLI prompt, trigger the review (the skill automatically saves the complete audit to `reports/SECURITY_ASSESSMENT_REPORT.md`):

```text
Perform a full security review of the application, container, and infrastructure using the container-security-review skill.
```

*A detailed analysis of this review is documented in [`reports/SECURITY_ASSESSMENT_REPORT.md`](./reports/SECURITY_ASSESSMENT_REPORT.md).*

### 2. What the Agent Inspects

The agent autonomously walks through the 3-layer audit checklist:
- **Layer 3 — Platform:** Cloud Run ingress, unauthenticated invoker permissions (`allUsers`), IAM service account blast radius, and Secret Manager exposure.
- **Layer 2 — Container:** Base image bloat (`golang:1.26` vs. distroless/alpine), lack of `USER` directive (running as root), baked secrets in `ENV` image layers, and PID 1 signal handling.
- **Layer 1 — Application:** Fallback secrets in code, debug diagnostic endpoints dumping environment variables, error handling reflection leaks, and dependency integrity.

### 3. Interactive Hardening Walkthrough (Fix One by One)

Instead of having the agent patch everything automatically in a single batch, attendees work interactively layer by layer—**keeping you in full control of every code change**. You direct Antigravity as an interactive pair-programmer: the agent explains each finding and drafts the solution, but you inspect the diffs, understand the security trade-offs, and apply fixes deliberately:

#### Phase 1: Harden Layer 1 — Application Code (`application/`)
Prompt the agent to guide you through fixing the application-level logic flaws:
```text
Let's fix Layer 1 (Application) findings one by one. Guide me through:
1. Removing the hardcoded fallback admin key in application/internal/config/config.go.
2. Eliminating debug reflection leaks in HTTP error responses.
3. Disabling the unauthenticated /api/v1/system/diagnostics endpoint.
```
> **What attendees learn:** Secure configuration loading (failing closed when credentials are missing instead of falling back to insecure defaults), sanitizing error responses to prevent information disclosure, and eliminating debug diagnostic endpoints from production code.

#### Phase 2: Harden Layer 2 — Container Definition (`application/Dockerfile`)
Prompt the agent to guide you through restructuring the container build:
```text
Let's fix Layer 2 (Container). Walk me through:
1. Converting application/Dockerfile into a multi-stage distroless build (gcr.io/distroless/static-debian12).
2. Removing the baked ENV TODO_ADMIN_KEY layer secret.
3. Enforcing non-root execution with USER nonroot:nonroot (UID 65532).
```
> **What attendees learn:** Reducing attack surface from hundreds of megabytes to a minimal static binary, stripping OS package managers (`apt`, `dpkg`) and shells (`bash`, `sh`), eliminating baked layer secrets visible via `docker history`, and dropping root privileges.

#### Phase 3: Harden Layer 3 — Platform & Cloud Run (`infrastructure/`)
Prompt the agent to guide you through Terraform infrastructure changes:
```text
Let's fix Layer 3 (Platform) in infrastructure/cloud_run.tf. Walk me through:
1. Setting DEBUG="false" and ENVIRONMENT="production".
2. Reviewing the IAM service account permissions to ensure least privilege.
```
> **What attendees learn:** Preventing staging/debug flags from leaking into deployed cloud services and enforcing least-privilege service account scopes.

---

## Step 4: Beyond AI Review — Deterministic Measurement & Verification

> [!IMPORTANT]
> **Is AI Review Enough?**  
> **No.** AI is exceptional for **contextual judgement, architecture analysis, and finding logic flaws** (e.g., discovering that a diagnostic endpoint exposes secrets or that a service account has broad storage admin roles).  
> However, AI cannot replace **deterministic measurement**. AI cannot guess real-time vulnerability feeds, calculate cryptographic hashes, or verify exact CVE matches in binary symbol tables.  
> Production defense requires **AI Judgement + Deterministic Tooling**:
> - **`govulncheck`:** Deterministic AST analysis of Go source code to detect if vulnerable functions are actually called.
> - **`trivy`:** Deterministic container filesystem scanner against official CVE vulnerability databases (OS packages, binaries, and dependencies).

### 1. Deterministic Application Audit with `govulncheck`
Run Go's official vulnerability checker on the application code:

```bash
# Ensure reports directory exists
mkdir -p reports

cd application
go install golang.org/x/vuln/cmd/govulncheck@latest

# Output audit results directly to reports/APPLICATION_SCAN_REPORT.md
govulncheck ./... | tee ../reports/APPLICATION_SCAN_REPORT.md
cd ..
```
*`govulncheck` analyzes your call graph — it alerts you if your compiled binary actually calls known-vulnerable symbols in imported modules, avoiding false-positive noise.*

> [!NOTE]
> **Why doesn't `govulncheck` show severity ratings (CRITICAL / HIGH)?**  
> `govulncheck` intentionally avoids static CVSS severity ratings. In application source code, CVSS scores are often misleading—a "Critical" CVE in an imported library is harmless if your application never executes that symbol. `govulncheck` prioritizes **Call-Graph Reachability**: it separates vulnerabilities into **Called** (real attack vectors in your execution path) versus **Uncalled** (harmless dead code in required dependencies).

*A detailed analysis of this audit is documented in [`reports/APPLICATION_SCAN_REPORT.md`](./reports/APPLICATION_SCAN_REPORT.md).*

### 2. Deterministic Container Audit with `trivy`
Scan the active production/staging container image for OS-level and library vulnerabilities:

```bash
# Ensure reports directory exists
mkdir -p reports

# Output scan results directly to reports/CONTAINER_SCAN_REPORT.md
trivy image \
  --skip-version-check \
  --output reports/CONTAINER_SCAN_REPORT.md \
  "${REGION}-docker.pkg.dev/${PROJECT_ID}/todo-repo/todo-service:stable"

# Or export structured JSON for automated tooling
trivy image \
  --skip-version-check \
  --format json \
  --output reports/CONTAINER_SCAN_REPORT.json \
  "${REGION}-docker.pkg.dev/${PROJECT_ID}/todo-repo/todo-service:stable"
```
*A detailed analysis of this scan is documented in [`reports/CONTAINER_SCAN_REPORT.md`](./reports/CONTAINER_SCAN_REPORT.md).*

Compare against the hardened distroless image after applying fixes:

> [!NOTE]
> Run the following build **after** completing the Phase 2 Dockerfile hardening in Step 3.3 (converting to multi-stage distroless and dropping root):

```bash
# Build the hardened distroless image locally
docker build -t todo-service:hardened ./application

# Deterministically verify that CVEs and attack vectors drop to near zero
trivy image todo-service:hardened
```

### 3. Fix & Verify Loop
If deterministic tools flag vulnerabilities, follow a structured remediation loop:

1. **Remediate Application Dependencies (`application/go.mod`)**:
   - **Trace the Caller First**: Review the call traces in `govulncheck`. If a vulnerability is in a **transitive (indirect) dependency** (for example, `brotli` or `klauspost/compress` used inside Fiber), do not immediately bump that indirect package alone. Updating an indirect dependency directly can break compatibility with the library that imports it.
   - **Upgrade the Direct Dependency First**: Update the top-level module your app actually imports (e.g., Fiber):
     ```bash
     cd application
     go get -u github.com/gofiber/fiber/v3
     go mod tidy
     ```
     *Why?* The Fiber maintainers test and verify compatible versions of `brotli` and `compress`. Upgrading Fiber lets it safely pull in the tested, patched sub-dependencies.
   - **Pin the Transitive Dependency (Fallback)**: If the upstream direct dependency has not released a fix yet, you can explicitly pin the patched indirect library as a temporary workaround:
     ```bash
     go get -u github.com/andybalholm/brotli@v1.2.7
     go mod tidy
     go test ./...  # Always run tests to verify no breaking changes were introduced
     ```

2. **Remediate Container OS Packages & Base Image (`application/Dockerfile`)**:
   - Migrate to a multi-stage distroless base image (`gcr.io/distroless/static-debian12`) which strips the package manager, shell, and Debian system libraries.
   - **Pin Base Images Securely**: Avoid mutable tags like `:latest` which cause non-deterministic builds.
     - **Strict Immutability (Best Practice)**: Pin by cryptographic digest (e.g., `gcr.io/distroless/static-debian12@sha256:...`) to guarantee bit-for-bit reproducible builds and protect against supply chain tampering.
     - **Pragmatic Middle Ground**: If automated digest-updating tooling (like Renovate or Dependabot) is not yet integrated, pin down to exact semantic patch versions (e.g., `golang:1.24.1-alpine3.21` or specific release tags) instead of floating major tags, allowing controlled patch upgrades without build surprises.

3. **Re-verify Deterministically**:
   ```bash
   # Verify zero called symbols remain in Go code
   (cd application && govulncheck ./...)

   # Verify container CVEs drop to near zero
   trivy image todo-service:hardened
   ```

---

## Step 5: Shift-Left Security & Organization-Level Guardrails

> [!CAUTION]
> **Why Individual Knowledge & Local Control Are Not Enough:**  
> Relying solely on individual developers remembering to run Trivy, inspect Dockerfiles, or drop root users creates **inconsistent enforcement**. Teams face tight deadlines, human fatigue, and varying security maturity.  
> **Shift-Left Security** moves security checks as early as possible into the development lifecycle, while **Organization Policies** enforce non-negotiable guardrails at the cloud infrastructure level.

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                        SHIFT-LEFT & ORGANIZATION DEFENSE IN DEPTH                      │
├─────────────────────┬───────────────────────────┬──────────────────────────────────────┤
│ 1. Developer Loop   │ 2. CI/CD Gateways         │ 3. Cloud / Org Guardrails            │
│ (IDE & Pre-commit)  │ (Cloud Build / GitHub)    │ (GCP Org Policies & IAM)             │
├─────────────────────┼───────────────────────────┼──────────────────────────────────────┤
│ • Antigravity Agent │ • Automated govulncheck   │ • GCP Org Policies (disable public)  │
│ • Trivy IDE scan    │ • Trivy build blocker     │ • Disable SA JSON Key Creation       │
│ • Pre-commit hooks  │ • Artifact Reg CVE scan   │ • VPC Service Controls               │
│ • Dockerfile linter │ • Automated PR checks     │ • Least-privilege IAM guardrails     │
└─────────────────────┴───────────────────────────┴──────────────────────────────────────┘
```

### 1. Shift-Left CI/CD Pipeline Gates & Automated Reporting
Automate security gates on every pull request using GitHub Actions. Insecure code or vulnerable containers are blocked before merging, and comprehensive audit reports are automatically published to the **GitHub Job Summary**, stored as **downloadable artifacts**, and integrated into the **GitHub Security (SARIF)** dashboard:

```yaml
# .github/workflows/ci.yml
name: CI & Security Gates

on:
  pull_request:
    branches: [main, master]
  push:
    branches: [main, master]

jobs:
  # =============================================================================
  # Job 1: Unit Tests & Code Correctness
  # =============================================================================
  unit-tests:
    name: Unit Tests & Code Quality
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: 'application/go.mod'

      - name: Run Unit Tests with Race Detector
        run: |
          (cd application && go test -v -race ./...)

  # =============================================================================
  # Job 2: Shift-Left Vulnerability Gates & Security Reporting
  # =============================================================================
  security-gates:
    name: Shift-Left Security Gates
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write # Required for GitHub Security tab SARIF upload

    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Create Reports Directory
        run: mkdir -p reports

      # Gate 2: Application Vulnerability Gate (Call-graph reachability)
      - name: Run govulncheck & Save Report
        id: govulncheck
        continue-on-error: true
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          (cd application && govulncheck ./...) | tee reports/APPLICATION_SCAN_REPORT.md

      # Gate 3: Container Image Vulnerability Gate
      - name: Build Candidate Container Image
        run: |
          docker build -t todo-service:candidate ./application

      - name: Run Trivy Scanner (Generate Table Report)
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'todo-service:candidate'
          format: 'table'
          output: 'reports/CONTAINER_SCAN_REPORT.md'
          severity: 'CRITICAL,HIGH'

      - name: Run Trivy Scanner (Generate SARIF for GitHub Security)
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'todo-service:candidate'
          format: 'sarif'
          output: 'reports/trivy-results.sarif'
          severity: 'CRITICAL,HIGH'

      # =========================================================================
      # Automated Reporting (Always runs, even when vulnerabilities are found)
      # =========================================================================
      - name: Publish Audit Summary to GitHub Actions Job Summary
        if: always()
        run: |
          echo "## Security Audit Summary" >> $GITHUB_STEP_SUMMARY
          echo "### 1. Application Vulnerabilities (govulncheck)" >> $GITHUB_STEP_SUMMARY
          echo '```text' >> $GITHUB_STEP_SUMMARY
          cat reports/APPLICATION_SCAN_REPORT.md >> $GITHUB_STEP_SUMMARY 2>/dev/null || true
          echo '```' >> $GITHUB_STEP_SUMMARY
          echo "### 2. Container OS & Dependencies (Trivy)" >> $GITHUB_STEP_SUMMARY
          echo '```text' >> $GITHUB_STEP_SUMMARY
          cat reports/CONTAINER_SCAN_REPORT.md >> $GITHUB_STEP_SUMMARY 2>/dev/null || true
          echo '```' >> $GITHUB_STEP_SUMMARY

      - name: Upload Security Reports as Artifacts
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: security-audit-reports
          path: reports/

      - name: Upload SARIF to GitHub Security Tab
        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: 'reports/trivy-results.sarif'
          category: 'trivy-container'

      # =========================================================================
      # Gate Enforcement: Fails the build if either scanner detected vulnerabilities
      # =========================================================================
      - name: Enforce Security Gates (Block Merge on Findings)
        run: |
          FAILED=0
          if [ "${{ steps.govulncheck.outcome }}" != "success" ]; then
            echo "[FAILED] Gate 1: govulncheck detected vulnerable called symbols in Go code."
            FAILED=1
          fi
          if grep -q "CRITICAL" reports/CONTAINER_SCAN_REPORT.md || grep -q "HIGH" reports/CONTAINER_SCAN_REPORT.md; then
            echo "[FAILED] Gate 2: Trivy detected CRITICAL/HIGH vulnerabilities in container image."
            FAILED=1
          fi
          if [ $FAILED -ne 0 ]; then
            echo "[MERGE BLOCKED] Unmitigated security vulnerabilities found. Resolve them to pass CI."
            exit 1
          fi
          echo "[PASSED] All security gates passed cleanly."
```

### 2. Ecosystem Tooling Spectrum (Industry Options)
While this workshop uses lightweight open-source tools (`govulncheck`, `trivy`), enterprise organizations integrate a variety of specialized commercial and open-source tooling across each tier:

| Operational Layer | Tool Category | Tooling Examples | Primary Function |
|---|---|---|---|
| **Application Layer** | **SAST** (Static Application Security Testing) | **Synopsys Black Duck Polaris**, SonarQube, Semgrep, Checkmarx | Scans proprietary source code for logic vulnerabilities, injection flaws, and insecure patterns without executing the code. |
| **Application Layer** | **SCA** (Software Composition Analysis) | **Black Duck**, Snyk Open Source, `govulncheck`, Mend | Identifies open-source dependencies, tracks licensing compliance, and maps known CVEs across transitive libraries. |
| **Application & API** | **DAST** (Dynamic Application Security Testing) | **OWASP ZAP**, **Burp Suite Enterprise**, StackHawk, Invicti | Tests running live HTTP services black-box from the outside, attacking endpoints to find auth bypasses, injection, and header misconfigurations. |
| **Container Layer** | **Image Scanning** | **JFrog Xray**, **Aqua Image Scanner**, **Trivy**, Snyk Container, Harbor Scanner | Dissects container image layers, base OS packages (Debian/Alpine), and embedded language binaries for published CVEs. |
| **Container & Runtime** | **Runtime Protection & Admission** | **Aqua MicroEnforcer**, **Aqua KubeEnforcer / Trivy Operator**, Falco | Injects runtime behavioural profiling into serverless/container runtimes, blocking unexpected processes, shell spawning, or unauthorized syscalls. |

### 3. Organization-Level Cloud Policies (Non-Bypassable Guardrails)
At the GCP organization or folder level, prevent insecure configurations from being provisioned, regardless of who writes the Terraform code:

| Organization Constraint | Security Purpose | What it Blocks |
|---|---|---|
| `constraints/run.allowedIngress` | Enforce ingress boundary | Blocks deployment of Cloud Run services with `all` public ingress unless routed through Cloud Load Balancer or internal VPC. |
| `constraints/compute.vmExternalIpAccess` | Zero egress public IPs | Prevents accidental public IP assignments on serverless VPC connectors. |
| `constraints/iam.disableServiceAccountKeyCreation` | Prevent credential theft | Blocks creation of downloadable service account JSON keys (eliminates #1 credential leak vector). |

---

## Step 6: Cleanup

### 1. Destroy Terraform-Managed Resources
```bash
cd infrastructure
terraform destroy \
  -var="project_id=$PROJECT_ID" \
  -var="region=$REGION"
cd ..
```

> [!TIP]
> **Non-Empty Artifact Registry Repositories:**  
> If `terraform destroy` encounters an error deleting `todo-repo` because container images are still stored inside it, you can delete the repository via `gcloud artifacts repositories delete todo-repo --location="$REGION" --quiet`, or simply proceed to delete the entire project below.

### 2. Delete the GCP Project (Recommended Complete Teardown)
To wipe all container images, secrets, Cloud Build history, and guarantee zero ongoing charges:
```bash
gcloud projects delete "$PROJECT_ID" --quiet
```
