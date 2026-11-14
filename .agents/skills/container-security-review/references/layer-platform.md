# Layer 3 — Platform Security Checks

Covers: Cloud Run, IAM, Artifact Registry, Networking, Binary Authorization.
Adapt GCP-specific commands to your cloud provider (AWS ECS, Azure Container Apps, etc.).

---

## A01 — Broken Access Control + SSRF

**Checks:**
- Is Cloud Run publicly accessible with `--allow-unauthenticated` on sensitive routes?
- Is the service account over-permissioned (e.g., `roles/editor` or `roles/owner`)?
- Can the app reach the GCP metadata server (`169.254.169.254`) to steal SA tokens?
- Is there a VPC Service Control perimeter?

```bash
# Check invocation policy
gcloud run services describe <service> --format="json" | jq '.spec.template.metadata.annotations'

# Check SA permissions
gcloud projects get-iam-policy <project> --format=json | jq '.bindings[] | select(.members[] | contains("serviceAccount"))'

# Test SSRF to metadata (from inside container)
curl -sH "Metadata-Flavor: Google" http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token
```

**Fix:**
- Add Cloud Run invoker IAM to restrict access
- Use a dedicated, least-privilege service account
- Block metadata endpoint egress via VPC egress rules or organization policy `constraints/compute.vmExternalIpAccess`

---

## A02 — Security Misconfiguration

**Checks:**
- Is ingress set to `all` (public) instead of `internal` or `internal-and-cloud-load-balancing`?
- Is Cloud Armor (WAF) configured?
- Are CORS headers wildcard (`Access-Control-Allow-Origin: *`)?
- Is the Cloud Run timeout unreasonably long (allowing slow-loris attacks)?

```bash
gcloud run services describe <service> --format="value(spec.template.metadata.annotations.'run.googleapis.com/ingress')"
```

**Fix:**
```bash
gcloud run services update <service> --ingress=internal-and-cloud-load-balancing
```

---

## A03 — Software Supply Chain Failures

**Checks:**
- Is Artifact Registry vulnerability scanning enabled?
- Is Binary Authorization enforced (only signed images can deploy)?
- Are base images pulled from trusted, official registries?
- Is there a Cloud Build pipeline with provenance attestation?

```bash
# Check if scanning is enabled on the registry
gcloud artifacts repositories describe <repo> --location=<region> --format=json | jq '.vulnerabilityScanning'

# List Binary Authorization policy
gcloud container binauthz policy export
```

**Fix:**
- Enable Artifact Registry scanning: `gcloud artifacts repositories update <repo> --enable-vulnerability-scanning`
- Enforce Binary Authorization with an `ALLOW_ALL` → `REQUIRE_ATTESTATION` policy upgrade
- Use Cloud Build with SLSA provenance

---

## A04 — Cryptographic Failures

**Checks:**
- Are secrets stored as plain Cloud Run env vars instead of Secret Manager mounts?
- Is Artifact Registry using CMEK (Customer-Managed Encryption Keys)?
- Is HTTP traffic (non-TLS) allowed to reach the service?

```bash
# List env vars on service (look for secrets)
gcloud run services describe <service> --format=json | jq '.spec.template.spec.containers[].env'

# Check for secret refs vs plain values
# Plain value: {"name": "TODO_ADMIN_KEY", "value": "dev-test-..."}  ← BAD
# Secret ref:  {"name": "TODO_ADMIN_KEY", "valueFrom": {...}}       ← GOOD
```

**Fix:**
```bash
# Move secret to Secret Manager
echo -n "my-prod-key" | gcloud secrets create todo-admin-key --data-file=-

# Grant SA access
gcloud secrets add-iam-policy-binding todo-admin-key \
  --member="serviceAccount:<SA>@<PROJECT>.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Update Cloud Run to use secret ref
gcloud run services update <service> \
  --update-secrets=TODO_ADMIN_KEY=todo-admin-key:latest \
  --remove-env-vars=TODO_ADMIN_KEY
```

---

## A06 — Insecure Design

**Checks:**
- Are Cloud Run CPU/memory limits set appropriately (not unlimited)?
- Is there network segmentation (VPC connector, private services)?
- Is there a Cloud Armor rate-limiting policy?
- Are multiple services sharing one Cloud Run instance with no isolation?

```bash
gcloud run services describe <service> --format=json | jq '.spec.template.spec.containers[].resources'
```

---

## A07 — Authentication Failures

**Checks:**
- Is Cloud Run protected by IAP (Identity-Aware Proxy) or Cloud Run Invoker IAM?
- Are there any unauthenticated admin routes?
- Is token/session expiry configured?

```bash
# Check if service allows unauthenticated invocations
gcloud run services get-iam-policy <service> --region=<region>
# If allUsers has roles/run.invoker → publicly invocable ← flag this
```

**Fix:**
```bash
# Remove public access
gcloud run services remove-iam-policy-binding <service> \
  --member="allUsers" --role="roles/run.invoker" --region=<region>
```

---

## A08 — Software & Data Integrity Failures

**Checks:**
- Is Binary Authorization enforced before deployment?
- Is there a Cloud Build trigger that validates image signatures (Cosign/Sigstore)?
- Is there an SBOM generated and stored for each build?

---

## A09 — Security Logging & Alerting Failures

**Checks:**
- Are Cloud Audit Logs enabled for Data Access on Secret Manager and Cloud Run?
- Are log-based alerts configured for suspicious patterns (auth failures, 5xx spikes)?
- Is Cloud Run sending structured JSON logs (parseable by Cloud Logging)?

```bash
# Check audit log config
gcloud projects get-iam-policy <project> --format=json | jq '.auditConfigs'
```

**Fix:**
- Enable Data Access audit logs for `run.googleapis.com` and `secretmanager.googleapis.com`
- Create log-based metrics + alerts for `resource.type="cloud_run_revision" AND severity>=ERROR`

---

## A10 — Mishandling of Exceptional Conditions

**Checks:**
- Is Cloud Run timeout set too high (allowing hung requests to consume quota)?
- Is there a retry/circuit-breaker policy for downstream dependencies?
- Are 5xx errors surfacing internal details to callers?

```bash
gcloud run services describe <service> --format="value(spec.template.spec.timeoutSeconds)"
```
