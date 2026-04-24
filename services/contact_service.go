package services

import (
	"context"
	"errors"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/turnstile"
	"github.com/zatrano/framework/repositories"
	"github.com/zatrano/framework/requests"
)

type IContactService interface {
	Submit(ctx context.Context, req requests.ContactSubmitRequest, ip, userAgent string) error
	ListMessages(ctx context.Context, params requests.ContactMessageListParams) (*requests.PaginatedResult, error)
	GetMessage(ctx context.Context, id uint) (*models.ContactMessage, error)
	MarkMessageRead(ctx context.Context, id uint) error
}

type ContactService struct {
	repo repositories.IContactRepository
}

func NewContactService(repo repositories.IContactRepository) IContactService {
	return &ContactService{repo: repo}
}

// Submit iletişim formunu işler: Turnstile doğrula, validasyon, DB'ye kaydet.
func (s *ContactService) Submit(ctx context.Context, req requests.ContactSubmitRequest, ip, userAgent string) error {
	if errs := req.Validate(); len(errs) > 0 {
		return errors.New("validasyon hatası")
	}

	siteKey := envconfig.String("TURNSTILE_SITE_KEY", "")
	secret := envconfig.String("TURNSTILE_SECRET_KEY", "")
	if siteKey != "" || secret != "" {
		if secret == "" {
			return errors.New("Turnstile yapılandırması eksik")
		}
		if req.TurnstileResponse == "" {
			return errors.New("Lütfen güvenlik doğrulamasını tamamlayın.")
		}
		resp, err := turnstile.Verify(secret, req.TurnstileResponse, ip)
		if err != nil {
			return err
		}
		if resp == nil || !resp.Success {
			return errors.New("İnsan doğrulaması başarısız. Lütfen tekrar deneyin.")
		}
	}

	msg := &models.ContactMessage{
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Subject:   req.Subject,
		Message:   req.Message,
		IP:        ip,
		UserAgent: userAgent,
	}
	return s.repo.Create(ctx, msg)
}

func (s *ContactService) ListMessages(ctx context.Context, params requests.ContactMessageListParams) (*requests.PaginatedResult, error) {
	list, total, err := s.repo.ListPaginated(ctx, params)
	if err != nil {
		return nil, err
	}
	return requests.CreatePaginatedResult(list, total, params.Page, params.PerPage), nil
}

func (s *ContactService) GetMessage(ctx context.Context, id uint) (*models.ContactMessage, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ContactService) MarkMessageRead(ctx context.Context, id uint) error {
	return s.repo.MarkRead(ctx, id)
}
