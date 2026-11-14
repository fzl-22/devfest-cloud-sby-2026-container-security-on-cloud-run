---
name: container-security-review
description: >-
  Use this skill when the user asks to review, audit, or assess the security of
  a Dockerfile, container image, or Cloud Run service. It guides the agent through
  a structured security review across three layers (Platform, Container/Runtime,
  Application) using the OWASP Top 10:2025 as the assessment framework. Applicable
  to any containerized workload, not just this project.
---

# Container Security Review

A structured security audit across **three layers**, mapped to **OWASP Top 10:2025**.

```
┌─────────────────────────────────────────────┐
│  LAYER 3 — PLATFORM                         │
│  Cloud Run · IAM · Networking · Registry     │
├─────────────────────────────────────────────┤
│  LAYER 2 — CONTAINER / RUNTIME              │
│  Dockerfile · Image · Runtime Config         │
├─────────────────────────────────────────────┤
│  LAYER 1 — APPLICATION                      │
│  Source Code · Config · Secrets · Logging    │
└─────────────────────────────────────────────┘
```

Each OWASP category is examined at all three layers.
Findings include: **evidence → risk → fix**.

---

## OWASP Top 10:2025 — At a Glance

| # | Category | Key Container Concern |
|---|---|---|
| A01 | Broken Access Control + SSRF | Root user, SSRF to metadata endpoint, unauthenticated endpoints |
| A02 | Security Misconfiguration | No `USER`, debug in prod, PID 1, open ports |
| A03 | Software Supply Chain Failures | Unpinned images, no signing, unscanned deps |
| A04 | Cryptographic Failures | Baked secrets, weak keys, plaintext traffic |
| A05 | Injection | Shell-form ENTRYPOINT, unvalidated env vars in commands |
| A06 | Insecure Design | No resource limits, no network segmentation, no rate limiting |
| A07 | Authentication Failures | Hardcoded keys, no token expiry, unauthenticated Cloud Run |
| A08 | Software & Data Integrity Failures | No image digest pinning, no SBOM, no provenance |
| A09 | Security Logging & Alerting Failures | No structured logs, secrets in logs, no audit trail |
| A10 | Mishandling of Exceptional Conditions | PID 1 signal handling, panic stack traces, no graceful shutdown |

---

## Review Process

### Step 1 — Gather Artifacts

Collect what is available:
- [ ] `Dockerfile`
- [ ] `go.mod` / `package.json` / `requirements.txt` (dependency manifest)
- [ ] Application source code entry point
- [ ] Cloud Run service configuration (`gcloud run services describe`)
- [ ] Runtime environment dump (env vars, debug endpoints)
- [ ] `docker exec` access or image inspection output

### Step 2 — Layer-by-Layer Review

Work through each layer's checklist:

- **Platform layer**: [references/layer-platform.md](./references/layer-platform.md)
- **Container/Runtime layer**: [references/layer-container.md](./references/layer-container.md)
- **Application layer**: [references/layer-application.md](./references/layer-application.md)

### Step 3 — Runtime Verification

Where possible, verify findings at runtime:

```bash
# Confirm running user
docker exec <container> id

# Confirm shell exists (should not in distroless)
docker exec <container> /bin/sh

# Confirm compiler / build tools in image (bad)
docker exec <container> go version
docker exec <container> which gcc

# Confirm source code in image (bad)
docker exec <container> ls /app

# Confirm env secrets
docker exec <container> printenv | grep -iE "key|secret|token|password"

# Confirm SSRF to GCP metadata server
curl -H "Metadata-Flavor: Google" http://169.254.169.254/computeMetadata/v1/
```

### Step 4 — Generate the Report

Produce a findings table:

| # | Layer | Finding | OWASP | Risk | Fix |
|---|---|---|---|---|---|
| 1 | Container | Runs as root (no `USER`) | A02 | Privilege escalation | Add `USER nonroot` or use distroless `:nonroot` |
| 2 | Container | Baked secret in `ENV` | A04 | Secret exposure | Remove; use Secret Manager |
| ... | ... | ... | ... | ... | ... |

Then provide:
1. **Prioritized fix list** (Critical → High → Medium → Low)
2. **Fixed Dockerfile** — see [references/fixed-dockerfile-template.md](./references/fixed-dockerfile-template.md)
3. **Platform remediation steps** where applicable
