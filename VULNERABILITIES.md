# Intentional Vulnerabilities — Workshop Reference

This document lists every deliberate security flaw embedded in this codebase for the
**Container Security on Cloud Run** workshop. These are real-world mistakes, not
contrived examples — each one has been observed in production systems.

> [!CAUTION]
> This repository is **intentionally insecure**. Do not use any of these patterns
> in real workloads. The flaws are documented here so attendees can find, understand,
> and fix them during the workshop.

---

## Layer 1 — Application (`application/`)

### V-APP-01 · Hardcoded fallback admin key in source code
- **File:** [`application/main.go`](../application/main.go) — `loadConfig()` function
- **OWASP:** A04 Cryptographic Failures / A07 Authentication Failures
- **Flaw:** When neither `APP_CONFIG_SECRETS` nor `TODO_ADMIN_KEY` env vars are set,
  the app falls back to a hardcoded key (`dev-test-admin-secret-key-3b9e4a1`).
  Teams often commit this so integration tests pass without a live Secret Manager.
- **Risk:** Any deployment without proper secret injection silently uses the known key.

### V-APP-02 · Debug mode leaks internal error details to callers
- **File:** [`application/main.go`](../application/main.go) — `HandleCreate()`
- **OWASP:** A02 Security Misconfiguration / A09 Security Logging & Alerting Failures
- **Flaw:** When `DEBUG=true`, binding/validation errors return raw Go reflection
  internals and the full request body to the HTTP caller.
- **Risk:** Reveals internal struct layout, field names, and raw payloads to attackers.

### V-APP-03 · Unauthenticated diagnostic endpoint enabled in staging
- **File:** [`application/main.go`](../application/main.go) — `HandleDiagnostics()`
- **OWASP:** A02 Security Misconfiguration / A01 Broken Access Control
- **Flaw:** A `GET /api/v1/system/diagnostics` endpoint dumps all environment variables
  (including secrets) when `DEBUG=true`. It was added to troubleshoot Cloud Run
  cold-starts and left enabled because it was "guarded by a flag".
- **Risk:** Full secret exposure — `TODO_ADMIN_KEY`, `APP_CONFIG_SECRETS`, and all
  other env vars are readable by anyone who can reach the service URL.

---

## Layer 2 — Container (`application/Dockerfile`)

### V-CON-01 · Full Golang image used as runtime (no multi-stage build)
- **File:** [`application/Dockerfile`](../application/Dockerfile) — `FROM golang:1.26`
- **OWASP:** A03 Software Supply Chain Failures / A02 Security Misconfiguration
- **Flaw:** The same `golang:1.26` image used to compile is also shipped to production.
  This includes the full Debian OS, `bash`, `apt`, the Go compiler, and all build tools.
- **Risk:** An attacker with RCE can open an interactive shell, install tools via `apt`,
  recompile a backdoored binary, and pivot to other services.

### V-CON-02 · Admin secret baked into Dockerfile ENV layer
- **File:** [`application/Dockerfile`](../application/Dockerfile) — `ENV TODO_ADMIN_KEY=...`
- **OWASP:** A04 Cryptographic Failures
- **Flaw:** `TODO_ADMIN_KEY=dev-test-admin-secret-key-3b9e4a1` is set via `ENV` in the
  Dockerfile "so it runs in Docker Desktop without needing extra config".
- **Risk:** The secret is permanently baked into the image layer and visible in
  `docker history` even if unset in a later layer. Anyone with registry pull access
  can read it.

### V-CON-03 · Container runs as root (no USER directive)
- **File:** [`application/Dockerfile`](../application/Dockerfile) — missing `USER`
- **OWASP:** A01 Broken Access Control
- **Flaw:** No `USER` directive means the process runs as UID 0 (root) inside the container.
- **Risk:** Any exploit immediately has root-level access to the container filesystem,
  all env vars, and potential host escape vectors.

---

## Layer 3 — Platform (`infrastructure/`)

### V-PLT-01 · DEBUG=true deployed to staging/production
- **File:** [`infrastructure/cloud_run.tf`](../infrastructure/cloud_run.tf) — `DEBUG` env var
- **OWASP:** A02 Security Misconfiguration
- **Flaw:** `DEBUG=true` is hardcoded in the Terraform Cloud Run service definition,
  enabling the diagnostic endpoint (V-APP-03) and verbose error responses (V-APP-02)
  in a non-development environment.
- **Risk:** Activates all debug-gated vulnerabilities in staging/production.

### V-PLT-02 · Secrets bundled as a JSON blob in a single Secret Manager secret
- **File:** [`infrastructure/cloud_run.tf`](../infrastructure/cloud_run.tf) — `APP_CONFIG_SECRETS` env var
- **File:** [`infrastructure/secrets.tf`](../infrastructure/secrets.tf) — `todo_secrets_v1`
- **OWASP:** A04 Cryptographic Failures / A02 Security Misconfiguration
- **Flaw:** Multiple credentials (database credentials, JWT signing key, admin API key, SaaS tokens) are bundled into one JSON blob and mounted as a single environment variable (`APP_CONFIG_SECRETS`).
- **Risk:** High blast radius. A single point of compromise exposes every system credential at once (database, auth, external APIs). Environment variables are also visible via `/proc/1/environ`, memory dumps, and the diagnostics endpoint.

---

## Attack Chain (Workshop Demo)

The flaws above chain together into a complete exploit:

```
V-PLT-01 (DEBUG=true in Terraform)
    → enables V-APP-03 (diagnostics endpoint publicly accessible)
        → exposes V-PLT-02 (APP_CONFIG_SECRETS JSON blob with DB, JWT, and Admin keys)
            → extracts V-APP-01 (admin key)
                → allows authenticated DELETE of any todo item
```

```bash
# Full exploit in one command
KEY=$(curl -s "$SERVICE_URL/api/v1/system/diagnostics" | jq -r '.environment_variables.APP_CONFIG_SECRETS | fromjson | .admin_api_key')

curl -i -X DELETE "$SERVICE_URL/api/v1/todos/1" -H "X-Admin-Key: $KEY"
```

---

## Fix Summary

| ID | Fix |
|---|---|
| V-APP-01 | Remove hardcoded fallback key; fail fast if secret not injected |
| V-APP-02 | Never return raw error internals to callers; log server-side only |
| V-APP-03 | Remove or protect the endpoint behind authentication; never rely on a debug flag |
| V-CON-01 | Multi-stage build: compile in `golang:1.26`, ship with `distroless/static-debian12` |
| V-CON-02 | Remove `ENV TODO_ADMIN_KEY=...`; inject via Secret Manager at runtime |
| V-CON-03 | Add `USER nonroot:nonroot` or use `distroless:nonroot` variant |
| V-PLT-01 | Set `DEBUG=false` for staging/production in Terraform |
| V-PLT-02 | Store each secret separately in Secret Manager; mount as individual secret refs |
