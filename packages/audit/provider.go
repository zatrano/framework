package audit

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "audit",
		Key:         "audit",
		Description: "Request/audit event log",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the audit addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	memoryAudit := NewMemoryStore(500)
	fileAudit, err := NewFileStore(app.BasePath("storage", "logs", "audit.jsonl"))
	var mgr *Manager
	if err != nil {
		if app.Logger() != nil {
			app.Logger().Errorf("audit store: %v", err)
		}
		mgr = New(memoryAudit)
	} else {
		mgr = New(&teeAuditStore{primary: memoryAudit, secondary: fileAudit})
	}
	app.Container().Instance("audit", mgr)
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }

type teeAuditStore struct {
	primary   Store
	secondary Store
}

func (s *teeAuditStore) Write(event Event) error {
	if err := s.primary.Write(event); err != nil {
		return err
	}
	return s.secondary.Write(event)
}

func (s *teeAuditStore) Recent(limit int) ([]Event, error) {
	return s.primary.Recent(limit)
}
