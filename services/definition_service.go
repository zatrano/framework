package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/repositories"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type IDefinitionService interface {
	GetValue(ctx context.Context, key, fallback string) string
	// GetMap, aktif tüm tanımları (definitions) key -> value sözlüğü olarak döner; kaynak yalnızca veritabanıdır.
	// Ortak middleware her istekte çağırdığı için kısa süreli bellek önbelleği kullanılır (DEFINITION_CACHE_TTL_SECONDS).
	GetMap(ctx context.Context) map[string]string

	GetAllDefinitions(ctx context.Context, params requests.DefinitionListParams) (*requests.PaginatedResult, error)
	GetDefinitionByID(ctx context.Context, id uint) (*models.Definition, error)
	CreateDefinition(ctx context.Context, req requests.CreateDefinitionRequest) error
	UpdateDefinition(ctx context.Context, id uint, req requests.UpdateDefinitionRequest) error
	DeleteDefinition(ctx context.Context, id uint) error
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

func (s *DefinitionService) invalidateCache() {
	s.mu.Lock()
	s.cache = nil
	s.cacheAt = time.Time{}
	s.mu.Unlock()
}

func (s *DefinitionService) GetAllDefinitions(ctx context.Context, params requests.DefinitionListParams) (*requests.PaginatedResult, error) {
	list, total, err := s.repo.ListPaged(ctx, params)
	if err != nil {
		return nil, errors.New("tanımlar getirilirken bir hata oluştu")
	}
	return requests.CreatePaginatedResult(list, total, params.Page, params.PerPage), nil
}

func (s *DefinitionService) GetDefinitionByID(ctx context.Context, id uint) (*models.Definition, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tanım bulunamadı")
		}
		return nil, errors.New("tanım bulunamadı")
	}
	return d, nil
}

func (s *DefinitionService) CreateDefinition(ctx context.Context, req requests.CreateDefinitionRequest) error {
	conv := req.BaseDefinitionRequest.Convert()
	if _, err := s.repo.GetByKeyAny(ctx, conv.Key); err == nil {
		return errors.New("bu anahtar zaten kullanılıyor")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("tanım oluşturulamadı")
	}

	d := &models.Definition{
		BaseModel: models.BaseModel{
			IsActive: conv.IsActive != nil && *conv.IsActive,
		},
		Key:         conv.Key,
		Value:       conv.Value,
		Description: conv.Description,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return errors.New("tanım oluşturulamadı")
	}
	s.invalidateCache()
	return nil
}

func (s *DefinitionService) UpdateDefinition(ctx context.Context, id uint, req requests.UpdateDefinitionRequest) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tanım bulunamadı")
		}
		return errors.New("tanım bulunamadı")
	}
	conv := req.BaseDefinitionRequest.Convert()
	if existing, err := s.repo.GetByKeyAny(ctx, conv.Key); err == nil && existing.ID != id {
		return errors.New("bu anahtar başka bir kayıt tarafından kullanılıyor")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("tanım güncellenemedi")
	}

	fields := map[string]interface{}{
		"key":         conv.Key,
		"value":       conv.Value,
		"description": conv.Description,
		"is_active":   conv.IsActive != nil && *conv.IsActive,
	}
	if err := s.repo.UpdateFields(ctx, id, fields); err != nil {
		return errors.New("tanım güncellenemedi")
	}
	s.invalidateCache()
	return nil
}

func (s *DefinitionService) DeleteDefinition(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.New("tanım silinemedi")
	}
	s.invalidateCache()
	return nil
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
