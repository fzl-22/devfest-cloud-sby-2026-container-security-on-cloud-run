# Todo Microservice (Go + Fiber)

This directory contains the application source code and container definition for the Todo REST API.

---

## Technical Stack

* **Language:** Go 1.26
* **Web Framework:** [Go Fiber v3](https://github.com/gofiber/fiber) (v3.3.0)
* **Default Port:** `8080` (configurable via `PORT` environment variable)

---

## Codebase Structure

Follows the standard Go project layout with internal packages:

```
application/
├── Dockerfile                  # Container build recipe
├── go.mod / go.sum             # Module dependencies
├── main.go                     # Entrypoint: wire config, store, handlers & HTTP server
└── internal/
    ├── config/                 # Runtime configuration & Secret Manager JSON parser
    ├── handler/                # Fiber HTTP request handlers, routes & diagnostics
    ├── model/                  # Domain types and request/response DTOs
    └── store/                  # In-memory thread-safe data store
```

* **`main.go`**: Lean entrypoint wiring together configuration, repository, and HTTP server middleware.
* **`internal/config`**: Loads `APP_CONFIG_SECRETS` (JSON from Secret Manager), `TODO_ADMIN_KEY`, with development fallback.
* **`internal/handler`**: Fiber HTTP controllers and route registration:
  * `GET /healthz`: Startup and liveness check.
  * `GET /api/v1/todos`: List all todos.
  * `GET /api/v1/todos/:id`: Get a specific todo.
  * `POST /api/v1/todos`: Create a new todo.
  * `DELETE /api/v1/todos/:id`: Delete a todo (Requires valid `X-Admin-Key` header).
  * `GET /api/v1/system/diagnostics`: System diagnostics endpoint (Active when `DEBUG=true`).
* **`internal/store`**: Thread-safe in-memory store (`sync.RWMutex`).
* **`internal/model`**: Domain structures (`Todo`, `CreateTodoRequest`, `UpdateTodoRequest`).
* **`Dockerfile`**: Single-stage container image based on `golang:1.26` building a static binary.

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
docker build -t todo-service:stable .
docker run -p 8080:8080 -e DEBUG=true todo-service:stable
```

### 3. Scan Image Locally
```bash
trivy image todo-service:stable
```
