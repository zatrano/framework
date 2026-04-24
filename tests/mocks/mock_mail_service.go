package mocks

import "github.com/stretchr/testify/mock"

// MockMailService — IMailService için test mock'u.
type MockMailService struct {
	mock.Mock
}

func (m *MockMailService) SendMail(to, subject, body string) error {
	return m.Called(to, subject, body).Error(0)
}

func (m *MockMailService) SendTemplateMail(to, subject, tmplName string, data map[string]interface{}) error {
	return m.Called(to, subject, tmplName, data).Error(0)
}
