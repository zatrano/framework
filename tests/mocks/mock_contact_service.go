package mocks

import (
	"context"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"github.com/stretchr/testify/mock"
)

type MockContactService struct {
	mock.Mock
}

func (m *MockContactService) Submit(ctx context.Context, req requests.ContactSubmitRequest, ip, ua string) error {
	return m.Called(ctx, req, ip, ua).Error(0)
}

func (m *MockContactService) ListMessages(ctx context.Context, params requests.ContactMessageListParams) (*requests.PaginatedResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*requests.PaginatedResult), args.Error(1)
}

func (m *MockContactService) GetMessage(ctx context.Context, id uint) (*models.ContactMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContactMessage), args.Error(1)
}

func (m *MockContactService) MarkMessageRead(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}
