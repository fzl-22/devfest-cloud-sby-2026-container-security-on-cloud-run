# Build and runtime container for the Todo API
#
# Looks like a standard developer Dockerfile:
# - Caches dependencies using separate COPY of go.mod / go.sum
# - Compiles static binary with CGO_ENABLED=0
# - Includes standard labels and expose directives
#
# But with real-world security gaps:
# 1. Base Image: Uses full 'golang:1.26' instead of multi-stage distroless (OWASP A06).
#    Ships full Debian OS, apt package manager, bash, and build utilities to production.
# 2. Baked Secret: Includes a fallback TODO_ADMIN_KEY in ENV layers (OWASP A02).
# 3. User: Missing USER directive; executes as UID 0 (root) (OWASP A01).

FROM golang:1.26

LABEL maintainer="devfest-surabaya"
LABEL description="Todo Microservice Backend"

WORKDIR /app

# Realistic flaw: A fallback administrative key placed in the Dockerfile
# "just so it runs in Docker Desktop without needing extra config".
ENV TODO_ADMIN_KEY=dev-test-admin-secret-key-3b9e4a1
ENV PORT=8080

# Cache module layers
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile statically
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/todo-service .

EXPOSE 8080

# Starts server directly as root
ENTRYPOINT ["/app/todo-service"]
