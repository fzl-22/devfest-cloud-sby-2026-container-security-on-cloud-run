package handler

import (
	"crypto/subtle"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"todo-application/internal/config"
	"todo-application/internal/model"
	"todo-application/internal/store"
)

type Handler struct {
	store  *store.TodoStore
	config *config.AppConfig
}

func NewHandler(s *store.TodoStore, cfg *config.AppConfig) *Handler {
	return &Handler{
		store:  s,
		config: cfg,
	}
}

// GET /api/v1/todos
func (h *Handler) HandleList(c fiber.Ctx) error {
	items := h.store.List()
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":  items,
		"total": len(items),
	})
}

// GET /api/v1/todos/:id
func (h *Handler) HandleGetByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id parameter must be an integer",
		})
	}

	todo, exists := h.store.GetByID(id)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("todo with id %d not found", id),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": todo})
}

// POST /api/v1/todos
func (h *Handler) HandleCreate(c fiber.Ctx) error {
	var req model.CreateTodoRequest
	if err := c.Bind().Body(&req); err != nil {
		// Log detailed error internally for Cloud Logging, never leak reflection to caller
		log.Printf("[WARN] JSON request binding failed: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid JSON request body",
		})
	}

	if strings.TrimSpace(req.Title) == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "title is a required field",
		})
	}

	todo := h.store.Create(req.Title, req.Description)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": todo})
}

// DELETE /api/v1/todos/:id (Administrative Action)
func (h *Handler) HandleDelete(c fiber.Ctx) error {
	// Check API key header
	authHeader := c.Get("X-Admin-Key")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required: missing X-Admin-Key header",
		})
	}

	// Security Hardening: Constant time comparison; fail closed if key unconfigured
	if h.config.AdminAPIKey == "" || subtle.ConstantTimeCompare([]byte(authHeader), []byte(h.config.AdminAPIKey)) != 1 {
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

	if err := h.store.Delete(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": fmt.Sprintf("todo item %d successfully deleted", id),
	})
}

// RegisterRoutes registers the application's HTTP routes onto the Fiber app.
func (h *Handler) RegisterRoutes(app *fiber.App) {
	// Public API group
	v1 := app.Group("/api/v1")
	{
		v1.Get("/todos", h.HandleList)
		v1.Get("/todos/:id", h.HandleGetByID)
		v1.Post("/todos", h.HandleCreate)
		v1.Delete("/todos/:id", h.HandleDelete)
	}
}
