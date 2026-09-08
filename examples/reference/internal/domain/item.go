package domain

import "strings"

// Item is a small domain record. Persistence details stay behind Repository.
type Item struct {
	ID   string
	Name string
}

// Repository is the persistence boundary. Implementations live in
// infrastructure (this reference uses an in-memory store).
type Repository interface {
	Get(id string) (Item, error)
	List() []Item
}

// ParseID rejects blank identifiers before they reach storage.
func ParseID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", ErrInvalidRequest
	}
	return id, nil
}
