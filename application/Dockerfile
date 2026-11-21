# ──────────────────────────────────────────────────────────────────────────────
# Stage 1: Build Environment
# Compiles a fully static binary; this build stage NEVER ships to production.
# ──────────────────────────────────────────────────────────────────────────────
FROM golang:1.27.4 AS builder

WORKDIR /app

# Cache dependency layer
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Compile statically with debugging symbols stripped (-w -s)
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o /app/todo-service .

# ──────────────────────────────────────────────────────────────────────────────
# Stage 2: Minimal Distroless Runtime
# No shell · No package manager · No compiler · Minimal attack surface
# ──────────────────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

LABEL maintainer="fzl-22"
LABEL description="Hardened Todo Microservice"

WORKDIR /app

# Safe non-secret runtime environment default
ENV PORT=8080

# Copy ONLY the compiled binary from the builder stage
COPY --from=builder /app/todo-service /app/todo-service

EXPOSE 8080

# Run as non-root user (UID 65532: nonroot)
USER nonroot:nonroot

# Exec-form entrypoint for correct PID 1 UNIX signal handling
ENTRYPOINT ["/app/todo-service"]
