package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// AppConfig represents parsed configuration loaded at runtime.
type AppConfig struct {
	Environment  string
	DebugMode    bool
	AdminAPIKey  string
	DatabaseHost string
}

type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTodoRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
}

type TodoStore struct {
	mu     sync.RWMutex
	todos  map[int]Todo
	nextID int
	config *AppConfig
}

// LoadConfig demonstrates realistic configuration loading logic in modern microservices:
// It looks for Secret Manager JSON payload first, then individual environment variables,
// but leaves a "staging fallback" that ends up deployed to production.
func LoadConfig() *AppConfig {
	cfg := &AppConfig{
		Environment:  os.Getenv("ENVIRONMENT"),
		DebugMode:    os.Getenv("DEBUG") == "true",
		DatabaseHost: os.Getenv("DB_HOST"),
	}

	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// 1. Try parsing JSON secret payload from Secret Manager (APP_CONFIG_SECRETS)
	if rawSecrets := os.Getenv("APP_CONFIG_SECRETS"); rawSecrets != "" {
		var secretMap map[string]string
		if err := json.Unmarshal([]byte(rawSecrets), &secretMap); err == nil {
			if key, ok := secretMap["admin_api_key"]; ok && key != "" {
				cfg.AdminAPIKey = key
			}
		} else {
			log.Printf("[WARN] Failed to parse APP_CONFIG_SECRETS JSON: %v", err)
		}
	}

	// 2. Fallback to standard environment variable
	if cfg.AdminAPIKey == "" {
		cfg.AdminAPIKey = os.Getenv("TODO_ADMIN_KEY")
	}

	if cfg.AdminAPIKey == "" {
		log.Println("[WARN] No ADMIN_API_KEY detected. Using local development fallback credential.")
		cfg.AdminAPIKey = "dev-test-admin-secret-key-3b9e4a1"
	}

	return cfg
}

func NewTodoStore(cfg *AppConfig) *TodoStore {
	store := &TodoStore{
		todos:  make(map[int]Todo),
		nextID: 1,
		config: cfg,
	}

	// Seed realistic initial items
	store.todos[1] = Todo{
		ID:          1,
		Title:       "Set up Cloud Run staging environment",
		Description: "Configure Terraform scripts and IAM permissions for the Todo service",
		Completed:   true,
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	}
	store.todos[2] = Todo{
		ID:          2,
		Title:       "Migrate runtime container to Distroless",
		Description: "Reduce attack surface by switching from golang:1.26 to distroless/static-debian12",
		Completed:   false,
		CreatedAt:   time.Now().Add(-12 * time.Hour),
		UpdatedAt:   time.Now().Add(-12 * time.Hour),
	}
	store.nextID = 3

	return store
}

// GET /api/v1/todos
func (s *TodoStore) HandleList(c fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Todo, 0, len(s.todos))
	for _, item := range s.todos {
		items = append(items, item)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":  items,
		"total": len(items),
	})
}

// GET /api/v1/todos/:id
func (s *TodoStore) HandleGetByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id parameter must be an integer",
		})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, exists := s.todos[id]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("todo with id %d not found", id),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": todo})
}

// POST /api/v1/todos
func (s *TodoStore) HandleCreate(c fiber.Ctx) error {
	var req CreateTodoRequest
	if err := c.Bind().Body(&req); err != nil {
		if s.config.DebugMode {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":        "payload validation failed",
				"debug_detail": err.Error(),
				"received_raw": string(c.Body()),
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid JSON request body",
		})
	}

	if strings.TrimSpace(req.Title) == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "title is a required field",
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	todo := Todo{
		ID:          s.nextID,
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.todos[s.nextID] = todo
	s.nextID++

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": todo})
}

// DELETE /api/v1/todos/:id (Administrative Action)
func (s *TodoStore) HandleDelete(c fiber.Ctx) error {
	// Check API key header
	authHeader := c.Get("X-Admin-Key")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required: missing X-Admin-Key header",
		})
	}

	// Constant time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(authHeader), []byte(s.config.AdminAPIKey)) != 1 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "insufficient privileges: invalid administrative key",
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id parameter must be an integer",
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.todos[id]; !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("todo with id %d does not exist", id),
		})
	}

	delete(s.todos, id)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": fmt.Sprintf("todo item %d successfully deleted", id),
	})
}

// GET /api/v1/system/diagnostics
func (s *TodoStore) HandleDiagnostics(c fiber.Ctx) error {
	if !s.config.DebugMode {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "endpoint not found",
		})
	}

	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			envVars[pair[0]] = pair[1]
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"system": fiber.Map{
			"environment":   s.config.Environment,
			"debug_enabled": s.config.DebugMode,
			"pid":           os.Getpid(),
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		},
		"environment_variables": envVars,
	})
}

func main() {
	config := LoadConfig()
	store := NewTodoStore(config)

	// Realistic Server Setup:
	// Includes standard industry middleware (CORS, Request Logger, Panic Recover).
	// But still contains the subtle operational flaw: Fiber defaults for server timeouts
	// (ReadTimeout, WriteTimeout, IdleTimeout) are all 0 (infinite), allowing slow clients
	// to hold serverless worker connections open.
	app := fiber.New(fiber.Config{
		AppName: "DevFest Todo Service v1.0",
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "X-Admin-Key"},
	}))

	// Healthcheck endpoint (probed by Cloud Run startup/liveness probe)
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "healthy",
			"uptime":  "running",
			"version": "1.0.0",
		})
	})

	// Public API group
	v1 := app.Group("/api/v1")
	{
		v1.Get("/todos", store.HandleList)
		v1.Get("/todos/:id", store.HandleGetByID)
		v1.Post("/todos", store.HandleCreate)
		v1.Delete("/todos/:id", store.HandleDelete)

		// Exposed diagnostic endpoint
		v1.Get("/system/diagnostics", store.HandleDiagnostics)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[INFO] Starting Todo Service on :%s (env: %s, debug: %t)", port, config.Environment, config.DebugMode)
	log.Fatal(app.Listen(":" + port))
}
