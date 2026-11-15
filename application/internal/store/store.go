package store

import (
	"fmt"
	"sync"
	"time"

	"todo-application/internal/model"
)

type TodoStore struct {
	mu     sync.RWMutex
	todos  map[int]model.Todo
	nextID int
}

func NewTodoStore() *TodoStore {
	s := &TodoStore{
		todos:  make(map[int]model.Todo),
		nextID: 1,
	}

	// Seed realistic initial items
	s.todos[1] = model.Todo{
		ID:          1,
		Title:       "Set up Cloud Run staging environment",
		Description: "Configure Terraform scripts and IAM permissions for the Todo service",
		Completed:   true,
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	}
	s.todos[2] = model.Todo{
		ID:          2,
		Title:       "Migrate runtime container to Distroless",
		Description: "Reduce attack surface by switching from golang:1.26 to distroless/static-debian12",
		Completed:   false,
		CreatedAt:   time.Now().Add(-12 * time.Hour),
		UpdatedAt:   time.Now().Add(-12 * time.Hour),
	}
	s.nextID = 3

	return s
}

func (s *TodoStore) List() []model.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.Todo, 0, len(s.todos))
	for _, item := range s.todos {
		items = append(items, item)
	}
	return items
}

func (s *TodoStore) GetByID(id int) (model.Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.todos[id]
	return item, exists
}

func (s *TodoStore) Create(title, description string) model.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := model.Todo{
		ID:          s.nextID,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.todos[s.nextID] = item
	s.nextID++
	return item
}

func (s *TodoStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.todos[id]; !exists {
		return fmt.Errorf("todo with id %d does not exist", id)
	}

	delete(s.todos, id)
	return nil
}
