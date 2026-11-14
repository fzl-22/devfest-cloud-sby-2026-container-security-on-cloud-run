# Layer 2 — Container / Runtime Security Checks

Covers: Dockerfile, image build, runtime configuration, image hygiene.

---

## A01 — Broken Access Control + SSRF

**Checks:**
- Does the runtime image contain `curl`, `wget`, or other HTTP tools that enable SSRF pivoting?
- Is the container network mode `host` (bypasses network namespace)?

```bash
docker exec <container> which curl wget
docker inspect <container> | jq '.[].HostConfig.NetworkMode'
```

**Fix:** Use distroless — no curl, no wget, no shell.

---

## A02 — Security Misconfiguration

**Checks:**

| Item | Command | Bad Result |
|---|---|---|
| Running as root | `docker exec <c> id` | `uid=0(root)` |
| No `USER` in Dockerfile | `grep USER Dockerfile` | No match |
| Writable root filesystem | `docker inspect <c> \| jq '.[].HostConfig.ReadonlyRootfs'` | `false` |
| Extra Linux capabilities | `docker inspect <c> \| jq '.[].HostConfig.CapAdd'` | Non-null |
| Privileged mode | `docker inspect <c> \| jq '.[].HostConfig.Privileged'` | `true` |
| Debug env in image | `docker exec <c> printenv DEBUG` | `true` |
| Unnecessary exposed ports | Check `EXPOSE` lines in Dockerfile | More than needed |

**Fix:**
```dockerfile
# In runtime stage
USER nonroot:nonroot

# Or use distroless nonroot variant (handles uid automatically)
FROM gcr.io/distroless/static-debian12:nonroot
```

```bash
# Read-only filesystem at runtime
docker run --read-only <image>
```

---

## A03 — Software Supply Chain Failures

**Checks:**
- Is the base image tag pinned to a digest (`@sha256:...`) or just a mutable tag (`:1.26`, `:latest`)?
- Is the image signed (Cosign/Sigstore)?
- Are there known CVEs in the base image?

```bash
# Check image digest
docker inspect <image> | jq '.[].RepoDigests'

# Scan for vulnerabilities
trivy image <image>
# or use Artifact Registry built-in scanning
```

**Fix:** Multi-stage build. Build in `golang:1.26`, copy only the binary to the runtime image.
Choose the runtime base image based on your security requirements:

| Image | Size | Shell | Compiler | Package Manager | Attack Surface |
|---|---|---|---|---|---|
| `golang:1.26` | ~850MB | ✅ bash | ✅ go | ✅ apt | Huge |
| `debian:slim` | ~80MB | ✅ sh | ❌ | ✅ apt | Medium |
| `alpine:3` | ~5MB | ✅ ash | ❌ | ✅ apk | Low–Medium |
| `distroless/static` | ~2MB | ❌ | ❌ | ❌ | Minimal |

> **Alpine** is a popular middle-ground — tiny and no compiler, but still has `sh` and `apk`,
> meaning an attacker with RCE can still install tools and get a shell.
> **Distroless** removes all of that; `docker exec` with a shell fails entirely.

```dockerfile
# Pin to digest instead of mutable tag
FROM golang:1.26@sha256:<digest> AS builder
FROM gcr.io/distroless/static-debian12@sha256:<digest>
```

---

## A04 — Cryptographic Failures

**Checks:**
- Are secrets in `ENV` or `ARG` layers?
- Do `docker history` layers reveal secrets?

```bash
# Inspect image history for secrets
docker history --no-trunc <image> | grep -iE "env|arg|secret|key|token|password"

# Check running env
docker exec <container> printenv | grep -iE "key|secret|token|password|admin"
```

> [!CAUTION]
> Secrets baked into `ENV` layers persist in image history **even if unset in a later layer**.
> The only fix is to never put secrets in the Dockerfile.

**Fix:** Remove all `ENV SECRET=...` from Dockerfile. Inject at runtime via Secret Manager.

---

## A05 — Injection

**Checks:**
- Is `ENTRYPOINT` / `CMD` in shell form (allows injection)?

```dockerfile
# SHELL FORM — BAD (runs via /bin/sh -c, subject to injection)
ENTRYPOINT ./todo-service $FLAGS

# EXEC FORM — GOOD (direct exec, no shell)
ENTRYPOINT ["/app/todo-service"]
```

**Fix:** Always use exec form (`["binary", "arg1"]`) for `ENTRYPOINT` and `CMD`.

---

## A06 — Insecure Design

**Checks:**
- Does the image have no resource constraints set at the Dockerfile level?
- Is there a `HEALTHCHECK` directive?
- Is the working directory world-writable?

```bash
docker inspect <container> | jq '.[].HostConfig.Memory'  # 0 = unlimited ← bad
grep HEALTHCHECK Dockerfile
```

**Fix:**
```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD ["/app/todo-service", "-healthcheck"]
```

```bash
# Set limits at runtime
docker run --memory=256m --cpus=0.5 <image>
```

---

## A07 — Authentication Failures

**Checks:**
- Does the image expose an unauthenticated admin port or management endpoint?
- Are default credentials present in the image?

```bash
docker exec <container> printenv | grep -iE "admin|default|password"
```

---

## A08 — Software & Data Integrity Failures

**Checks:**
- Is there a `.dockerignore` to prevent leaking `.git`, `.env`, secrets into the build context?
- Are dependencies verified during build (`go mod verify`)?

```bash
# Check for .dockerignore
cat .dockerignore

# Check what gets into build context
docker build --no-cache . 2>&1 | head -5
```

**Fix:**
```
# .dockerignore
.git
.env
*.env
*.pem
*.key
secrets/
```

```dockerfile
RUN go mod verify   # fails if modules were tampered with
```

---

## A09 — Security Logging & Alerting Failures

**Checks:**
- Are container logs going to stdout/stderr in structured JSON format?
- Are access logs enabled?

```bash
docker logs <container> | head -10
# Should be structured JSON, not raw text
```

---

## A10 — Mishandling of Exceptional Conditions

**Checks:**
- Is the app running as PID 1?
- Is `SIGTERM` handled for graceful shutdown?
- Does the container restart indefinitely on panic?

```bash
# PID 1 check
docker exec <container> cat /proc/1/status | grep Name
docker inspect <container> | jq '.[].State.Pid'
# If the app binary is pid 1 → flag it

# Check restart policy
docker inspect <container> | jq '.[].HostConfig.RestartPolicy'
```

**Fix — handle signals in Go:**
```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
// graceful shutdown logic
```

**Fix — use tini (non-distroless):**
```dockerfile
RUN apt-get install -y tini
ENTRYPOINT ["/usr/bin/tini", "--", "/app/todo-service"]
```

**Fix — distroless handles this correctly** when using exec-form ENTRYPOINT.
