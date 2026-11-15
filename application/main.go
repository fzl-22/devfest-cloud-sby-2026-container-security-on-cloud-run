package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"todo-application/internal/config"
	"todo-application/internal/handler"
	"todo-application/internal/store"
)

func main() {
	cfg := config.LoadConfig()
	todoStore := store.NewTodoStore()
	h := handler.NewHandler(todoStore, cfg)

	// Server setup with middleware
	// Operational flaw preserved: Fiber defaults for timeouts (ReadTimeout, WriteTimeout, IdleTimeout)
	// are 0 (infinite), allowing slow clients to hold serverless worker connections open.
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

	// Register API routes
	h.RegisterRoutes(app)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[INFO] Starting Todo Service on :%s (env: %s, debug: %t)", port, cfg.Environment, cfg.DebugMode)
	log.Fatal(app.Listen(":" + port))
}
