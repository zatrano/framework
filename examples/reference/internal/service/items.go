package service

import "github.com/zatrano/framework/v2/examples/reference/internal/domain"

// Items is the application service. It depends on domain.Repository, not on
// a concrete store.
type Items struct {
	repo domain.Repository
}

// New constructs Items.
func New(repo domain.Repository) *Items {
	return &Items{repo: repo}
}

// Get returns one item after validating the identifier.
func (s *Items) Get(rawID string) (domain.Item, error) {
	id, err := domain.ParseID(rawID)
	if err != nil {
		return domain.Item{}, err
	}
	item, err := s.repo.Get(id)
	if err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

// List returns all items.
func (s *Items) List() []domain.Item {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.List()
}
