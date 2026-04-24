package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/repositories"
)

type IDefinitionService interface {
	GetValue(ctx context.Context, key, fallback string) string
	// GetMap, aktif tüm tanımları (definitions) key -> value sözlüğü olarak döner; kaynak yalnızca veritabanıdır.
	// Ortak middleware her istekte çağırdığı için kısa süreli bellek önbelleği kullanılır (DEFINITION_CACHE_TTL_SECONDS).
	GetMap(ctx context.Context) map[string]string
}

type DefinitionService struct {
	repo repositories.IDefinitionRepository

	mu      sync.RWMutex
	cache   map[string]string
	cacheAt time.Time
}

func NewDefinitionService(repo repositories.IDefinitionRepository) IDefinitionService {
	return &DefinitionService{repo: repo}
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
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

func (s *DefinitionService) definitionCacheTTL() time.Duration {
	sec := envconfig.Int("DEFINITION_CACHE_TTL_SECONDS", 120)
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

func (s *DefinitionService) loadMapFromDB(ctx context.Context) (map[string]string, error) {
	defs, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(defs))
	for _, d := range defs {
		out[d.Key] = strings.TrimSpace(d.Value)
	}
	return out, nil
}

func (s *DefinitionService) GetMap(ctx context.Context) map[string]string {
	ttl := s.definitionCacheTTL()
	if ttl <= 0 {
		m, err := s.loadMapFromDB(ctx)
		if err != nil {
			return map[string]string{}
		}
		return m
	}

	now := time.Now()
	s.mu.RLock()
	if len(s.cache) > 0 && now.Sub(s.cacheAt) < ttl {
		c := cloneStringMap(s.cache)
		s.mu.RUnlock()
		return c
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	now = time.Now()
	if len(s.cache) > 0 && now.Sub(s.cacheAt) < ttl {
		return cloneStringMap(s.cache)
	}
	m, err := s.loadMapFromDB(ctx)
	if err != nil {
		return map[string]string{}
	}
	s.cache = m
	s.cacheAt = now
	return cloneStringMap(m)
}
