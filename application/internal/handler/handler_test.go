package handler

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"todo-application/internal/config"
	"todo-application/internal/store"
)

func setupTestApp() (*fiber.App, *config.AppConfig) {
	cfg := &config.AppConfig{
		Environment: "testing",
		DebugMode:   true,
		AdminAPIKey: "test-secret-key",
	}
	s := store.NewTodoStore()
	h := NewHandler(s, cfg)

	app := fiber.New()
	h.RegisterRoutes(app)

	return app, cfg
}

func TestHandleList(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/todos", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandleDelete_Auth(t *testing.T) {
	app, _ := setupTestApp()

	// Unauthorized (no header)
	req := httptest.NewRequest("DELETE", "/api/v1/todos/1", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 Unauthorized without header, got %d", resp.StatusCode)
	}

	// Forbidden (wrong key)
	req = httptest.NewRequest("DELETE", "/api/v1/todos/1", nil)
	req.Header.Set("X-Admin-Key", "wrong-key")
	resp, _ = app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 Forbidden with wrong key, got %d", resp.StatusCode)
	}

	// OK (valid key)
	req = httptest.NewRequest("DELETE", "/api/v1/todos/1", nil)
	req.Header.Set("X-Admin-Key", "test-secret-key")
	resp, _ = app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 OK with valid key, got %d", resp.StatusCode)
	}
}

func TestHandleDiagnostics_Disabled(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/system/diagnostics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Security Hardening: Diagnostic endpoint must return 404 (disabled)
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 Not Found for disabled diagnostics endpoint, got %d", resp.StatusCode)
	}
}

func TestHandleCreate(t *testing.T) {
	app, _ := setupTestApp()

	payload := []byte(`{"title":"Write Decoupled Code","description":"Clean standard Go pattern"}`)
	req := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("expected 201 Created, got %d", resp.StatusCode)
	}
}
