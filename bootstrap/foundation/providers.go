package foundation

import (
	"fmt"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel"
)

// Providers returns the full foundation stack in dependency order.
func Providers() []kernel.Provider {
	return []kernel.Provider{
		&KernelServiceProvider{},
		&DatabaseServiceProvider{},
		&CacheServiceProvider{},
		&EventsServiceProvider{},
		&QueueServiceProvider{},
		&AuthServiceProvider{},
		&LocalizationServiceProvider{},
		&ScheduleServiceProvider{},
		&HTTPClientServiceProvider{},
		&BroadcastingServiceProvider{},
		&NotificationServiceProvider{},
		&FilesystemServiceProvider{},
		&AssetsServiceProvider{},
		&ViewSessionServiceProvider{},
	}
}

// KernelProviders returns secure-by-default core boot (no DB/session/auth).
func KernelProviders() []kernel.Provider {
	return []kernel.Provider{&KernelServiceProvider{}}
}

type KernelServiceProvider struct{}

func (p *KernelServiceProvider) Register(app contracts.App) error {
	k, ok := app.(*kernel.Application)
	if !ok {
		return fmt.Errorf("kernel boot requires *kernel.Application")
	}
	return k.BootKernelServices()
}
func (p *KernelServiceProvider) Boot(app contracts.App) error { return nil }

type DatabaseServiceProvider struct{}

func (p *DatabaseServiceProvider) Register(app contracts.App) error {
	return BootDatabase(app)
}
func (p *DatabaseServiceProvider) Boot(app contracts.App) error { return nil }

type CacheServiceProvider struct{}

func (p *CacheServiceProvider) Register(app contracts.App) error {
	return BootCacheServices(app)
}
func (p *CacheServiceProvider) Boot(app contracts.App) error { return nil }

type EventsServiceProvider struct{}

func (p *EventsServiceProvider) Register(app contracts.App) error {
	return BootEventsServices(app)
}
func (p *EventsServiceProvider) Boot(app contracts.App) error { return nil }

type QueueServiceProvider struct{}

func (p *QueueServiceProvider) Register(app contracts.App) error {
	return BootQueueServices(app)
}
func (p *QueueServiceProvider) Boot(app contracts.App) error { return nil }

type AuthServiceProvider struct{}

func (p *AuthServiceProvider) Register(app contracts.App) error {
	return BootAuthServices(app)
}
func (p *AuthServiceProvider) Boot(app contracts.App) error { return nil }

type LocalizationServiceProvider struct{}

func (p *LocalizationServiceProvider) Register(app contracts.App) error {
	return BootLocalizationServices(app)
}
func (p *LocalizationServiceProvider) Boot(app contracts.App) error { return nil }

type ScheduleServiceProvider struct{}

func (p *ScheduleServiceProvider) Register(app contracts.App) error {
	return BootScheduleServices(app)
}
func (p *ScheduleServiceProvider) Boot(app contracts.App) error { return nil }

type HTTPClientServiceProvider struct{}

func (p *HTTPClientServiceProvider) Register(app contracts.App) error {
	return BootHTTPClientServices(app)
}
func (p *HTTPClientServiceProvider) Boot(app contracts.App) error { return nil }

type BroadcastingServiceProvider struct{}

func (p *BroadcastingServiceProvider) Register(app contracts.App) error {
	return BootBroadcastingServices(app)
}
func (p *BroadcastingServiceProvider) Boot(app contracts.App) error { return nil }

type NotificationServiceProvider struct{}

func (p *NotificationServiceProvider) Register(app contracts.App) error {
	return BootNotificationServices(app)
}
func (p *NotificationServiceProvider) Boot(app contracts.App) error { return nil }

type FilesystemServiceProvider struct{}

func (p *FilesystemServiceProvider) Register(app contracts.App) error {
	return BootFilesystemServices(app)
}
func (p *FilesystemServiceProvider) Boot(app contracts.App) error { return nil }

type AssetsServiceProvider struct{}

func (p *AssetsServiceProvider) Register(app contracts.App) error {
	return BootAssetsServices(app)
}
func (p *AssetsServiceProvider) Boot(app contracts.App) error { return nil }

type ViewSessionServiceProvider struct{}

func (p *ViewSessionServiceProvider) Register(app contracts.App) error {
	return BootViewSession(app)
}
func (p *ViewSessionServiceProvider) Boot(app contracts.App) error { return nil }
