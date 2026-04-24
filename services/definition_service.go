package services

import (
	"context"
	"strings"

	"github.com/zatrano/framework/repositories"
)

type IDefinitionService interface {
	GetValue(ctx context.Context, key, fallback string) string
	// GetMap, aktif tüm tanımları (definitions) key -> value sözlüğü olarak döner; kaynak yalnızca veritabanıdır.
	GetMap(ctx context.Context) map[string]string
}

type DefinitionService struct {
	repo repositories.IDefinitionRepository
}

func NewDefinitionService(repo repositories.IDefinitionRepository) IDefinitionService {
	return &DefinitionService{repo: repo}
}

func (s *DefinitionService) GetValue(ctx context.Context, key, fallback string) string {
	def, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fallback
	}
	value := strings.TrimSpace(def.Value)
	if value == "" {
		return fallback
	}
	return value
}

func (s *DefinitionService) GetMap(ctx context.Context) map[string]string {
	defs, err := s.repo.ListAll(ctx)
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(defs))
	for _, d := range defs {
		out[d.Key] = strings.TrimSpace(d.Value)
	}
	return out
}
