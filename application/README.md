# Todo Microservice (Go + Fiber)

This directory contains the application source code and container definition for the Todo REST API.

---

## Technical Stack

* **Language:** Go 1.26
* **Web Framework:** [Go Fiber v3](https://github.com/gofiber/fiber) (v3.3.0)
* **Default Port:** `8080` (configurable via `PORT` environment variable)

---

## Codebase Structure

* **`main.go`**:
  * Fiber application initialization with `logger`, `recover`, and `cors` middleware.
  * In-memory CRUD todo store (`sync.RWMutex`).
  * `LoadConfig()`: Loads `APP_CONFIG_SECRETS` (JSON from Secret Manager), `TODO_ADMIN_KEY`, with development fallback.
  * API Endpoints:
    * `GET /healthz`: Startup and liveness check.
    * `GET /api/v1/todos`: List all todos.
    * `GET /api/v1/todos/:id`: Get a specific todo.
    * `POST /api/v1/todos`: Create a new todo.
    * `DELETE /api/v1/todos/:id`: Delete a todo (Requires valid `X-Admin-Key` header).
    * `GET /api/v1/system/diagnostics`: System diagnostics endpoint (Active when `DEBUG=true`).
* **`Dockerfile`**:
  * Single-stage container image based on `golang:1.26`.
  * Builds a static Linux binary (`CGO_ENABLED=0`).
  * Contains baked-in `ENV TODO_ADMIN_KEY`.
* **`go.mod` & `go.sum`**: Pinned module dependencies.

---

## Local Development & Testing

### 1. Run Directly with Go
```bash
# Start server
go run main.go

# Start with debug mode enabled (activates /api/v1/system/diagnostics)
DEBUG=true go run main.go
```

Test endpoints:
```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/todos
curl http://localhost:8080/api/v1/system/diagnostics
```

### 2. Build Container Image Locally
```bash
docker build -t todo-service:v1 .
docker run -p 8080:8080 -e DEBUG=true todo-service:v1
```

### 3. Scan Image Locally
```bash
trivy image todo-service:v1
```
