package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
)

// Fallback admin key, used when ADMIN_API_KEY is not set.
// Flaw: the key lives in source control and ends up in the image.
const defaultAdminKey = "sk-live-7f3a9c2e41d8b6f0"

type Note struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

type notesStore struct {
	mu    sync.Mutex
	notes []Note
}

func (s *notesStore) list(c fiber.Ctx) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return c.Status(fiber.StatusOK).JSON(s.notes)
}

func (s *notesStore) create(c fiber.Ctx) error {
	var n Note
	if err := c.Bind().Body(&n); err != nil {
		// Flaw: internal error details go straight to the client.
		return c.Status(fiber.StatusBadRequest).SendString("bad request: " + err.Error())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n.ID = len(s.notes) + 1
	s.notes = append(s.notes, n)
	return c.Status(fiber.StatusCreated).JSON(n)
}

func (s *notesStore) delete(c fiber.Ctx) error {
	// "Admin" endpoint. Key comes from the env, with a hardcoded fallback.
	key := os.Getenv("ADMIN_API_KEY")
	if key == "" {
		key = defaultAdminKey
	}
	if c.Get("X-Admin-Key") != key {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	id := c.Params("id")
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.notes {
		if fmt.Sprint(n.ID) == id {
			s.notes = append(s.notes[:i], s.notes[i+1:]...)
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"deleted": id})
		}
	}
	return c.Status(fiber.StatusNotFound).SendString("not found")
}

// Debug endpoint left enabled in production: dumps every env var.
func (s *notesStore) env(c fiber.Ctx) error {
	out := map[string]string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return c.Status(fiber.StatusOK).JSON(out)
}

func main() {
	store := &notesStore{}

	app := fiber.New()
	app.Get("/notes", store.list)
	app.Post("/notes", store.create)
	app.Delete("/notes/:id", store.delete)
	app.Get("/env", store.env)
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("notes service v1")
	})

	// Flaw: no timeouts on the server — fiber's defaults are all zero,
	// so slow clients hold instances open.
	log.Println("listening on :8080")
	log.Fatal(app.Listen(":8080"))
}
