# OWASP Container Security Checks

Reference for the `container-security-review` skill.

---

## A01 — Broken Access Control: Running as Root

**Risk**: If the app is compromised, the attacker has full root access to the
container filesystem, can read all env vars, write anywhere, and potentially
escape to the host.

**Evidence**: Missing `USER` directive in Dockerfile; `whoami` returns `root`;
`HOME=/root` in env.

**Fix**:
```dockerfile
# In the runtime stage
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
USER nonroot:nonroot
```
Or with distroless (nonroot variant handles this automatically):
```dockerfile
FROM gcr.io/distroless/static-debian12:nonroot
```

---

## A02 — Cryptographic / Secret Failures: Baked Secrets

**Risk**: Secrets in `ENV` or `ARG` are baked into image layers. Anyone with
`docker history`, registry pull access, or who can exec into the container can
read them. They persist even if you unset them in a later layer.

**Evidence**: `ENV TODO_ADMIN_KEY=...` in Dockerfile; `APP_CONFIG_SECRETS` as
plaintext JSON env var.

**Fix**:
- Store secrets in **Google Cloud Secret Manager**
- Mount as files via Cloud Run secret volume mounts, not env vars
- Never use `ENV` or `ARG` for secrets in Dockerfiles

```yaml
# Cloud Run secret volume mount (preferred)
volumes:
  - name: secrets
    secret:
      secretName: todo-admin-key
```

---

## A05 — Security Misconfiguration: PID 1 & Debug Mode

### PID 1
**Risk**: The app becomes the init process. SIGTERM is not handled by default,
causing slow (or no) graceful shutdown. Cloud Run waits the full timeout then
SIGKILLs — wasting money and causing request drops.

**Fix**: Use `tini` as init, or handle signals explicitly in the app.
```dockerfile
RUN apk add --no-cache tini  # alpine example
ENTRYPOINT ["/sbin/tini", "--", "/app/todo-service"]
```
With distroless, use the `CMD` form and handle signals in Go with `signal.Notify`.

### Debug in Staging/Production
**Risk**: Debug endpoints, verbose logs, and stack traces expose internal
structure to attackers.

**Fix**: Set `DEBUG=false` for staging/production. Drive via environment-specific
Cloud Run environment variables, not baked into the image.

---

## A06 — Vulnerable and Outdated Components: Full Base Image

**Risk**: Shipping `golang:1.26` to production includes:
- Full Debian OS (~800MB)
- `apt-get` package manager (attackers can install tools)
- `bash`, `sh` (interactive shell for attackers)
- Go compiler + build tools (attackers can recompile backdoored binaries)
- All source code if `COPY . .` is used

**Fix**: Multi-stage build. Build in `golang:1.26`, copy only the binary to
`gcr.io/distroless/static-debian12`.

| Image | Size | Shell | Compiler | Attack Surface |
|---|---|---|---|---|
| `golang:1.26` | ~850MB | ✅ bash | ✅ go | Huge |
| `debian:slim` | ~80MB | ✅ sh | ❌ | Medium |
| `distroless/static` | ~2MB | ❌ | ❌ | Minimal |
