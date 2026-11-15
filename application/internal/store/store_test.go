package store

import (
	"testing"
)

func TestTodoStore_CRUD(t *testing.T) {
	s := NewTodoStore()

	// Initial seeds
	todos := s.List()
	if len(todos) != 2 {
		t.Fatalf("expected 2 seeded todos, got %d", len(todos))
	}

	// Create
	item := s.Create("New Task", "Task details")
	if item.ID != 3 || item.Title != "New Task" {
		t.Errorf("unexpected created item: %+v", item)
	}

	// GetByID
	fetched, exists := s.GetByID(3)
	if !exists || fetched.Title != "New Task" {
		t.Errorf("expected to find item 3, got %+v", fetched)
	}

	// Delete
	if err := s.Delete(3); err != nil {
		t.Fatalf("failed to delete item 3: %v", err)
	}

	// Delete non-existent
	if err := s.Delete(999); err == nil {
		t.Errorf("expected error deleting non-existent item, got nil")
	}
}
