# Fixed Dockerfile Template

A secure multi-stage Dockerfile for a Go service on Cloud Run.
Addresses A02, A03, A04, A10 at the Container layer.

```dockerfile
# ─────────────────────────────────────────────────────
# Stage 1: Build
# Pin to digest for supply chain integrity (A03/A08)
# This stage NEVER ships to production.
# ─────────────────────────────────────────────────────
FROM golang:1.26 AS builder
# Recommended: pin to digest
# FROM golang:1.26@sha256:<digest> AS builder

WORKDIR /app

# Cache dependency layer
COPY go.mod go.sum ./
RUN go mod download && go mod verify   # verify checksums (A08)

# Copy source and compile a fully static binary
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server .

# ─────────────────────────────────────────────────────
# Stage 2: Runtime (distroless)
# No shell · No compiler · No source · No package manager
# ─────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot
# Recommended: pin to digest
# FROM gcr.io/distroless/static-debian12@sha256:<digest>

LABEL org.opencontainers.image.source="https://github.com/your-org/your-repo"
LABEL org.opencontainers.image.vendor="your-org"

# PORT is not a secret — safe as ENV
ENV PORT=8080

# Copy ONLY the compiled binary (A03: no source code in runtime image)
COPY --from=builder /app/server /app/server

EXPOSE 8080

# :nonroot variant = uid=65532 (not root) — fixes A02
# Exec-form ENTRYPOINT = no shell injection, correct PID 1 signal handling — fixes A05/A10
ENTRYPOINT ["/app/server"]
```

## Security Properties of This Dockerfile

| Property | Before | After |
|---|---|---|
| Runtime user | `root` (uid=0) | `nonroot` (uid=65532) |
| Shell available | ✅ bash/sh | ❌ none |
| Go compiler in image | ✅ | ❌ |
| Source code in image | ✅ | ❌ |
| Apt/package manager | ✅ | ❌ |
| Image size | ~850 MB | ~5 MB |
| Baked secrets | ✅ ENV layer | ❌ removed |
| Dep integrity check | ❌ | ✅ `go mod verify` |
| PID 1 signal handling | ❌ broken | ✅ exec-form |

## Secret Injection (Platform Layer Fix)

Never use `ENV SECRET=...` in Dockerfile. Instead:

```bash
# 1. Store in Secret Manager
echo -n "my-prod-secret" | gcloud secrets create my-secret --data-file=-

# 2. Grant SA access
gcloud secrets add-iam-policy-binding my-secret \
  --member="serviceAccount:<SA>@<PROJECT>.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# 3. Mount in Cloud Run as a secret reference (not plain env var)
gcloud run services update <service> \
  --update-secrets=MY_SECRET=my-secret:latest
```

## Verification After Applying Fix

```bash
# Build
docker build -t myapp:secure .

# Run
docker run -d -p 8080:8080 --name myapp-test myapp:secure

# These should ALL FAIL — proves distroless is working
docker exec myapp-test /bin/sh       # ← exec: "/bin/sh": no such file
docker exec myapp-test whoami        # ← no such file
docker exec myapp-test go version    # ← no such file

# Image size
docker images | grep myapp
# myapp   secure   <5MB
# myapp   old      ~850MB

# Service still responds
curl http://localhost:8080/health
```
