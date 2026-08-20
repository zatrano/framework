package foundation

import "github.com/zatrano/framework/core"

// Providers returns the full foundation stack in dependency order.
func Providers() []core.Provider {
	return []core.Provider{
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
func KernelProviders() []core.Provider {
	return []core.Provider{&KernelServiceProvider{}}
}

type KernelServiceProvider struct{}

func (p *KernelServiceProvider) Register(app *core.Application) error {
	return app.BootKernelServices()
}
func (p *KernelServiceProvider) Boot(app *core.Application) error { return nil }

type DatabaseServiceProvider struct{}

func (p *DatabaseServiceProvider) Register(app *core.Application) error {
	return BootDatabase(app)
}
func (p *DatabaseServiceProvider) Boot(app *core.Application) error { return nil }

type CacheServiceProvider struct{}

func (p *CacheServiceProvider) Register(app *core.Application) error {
	return BootCacheServices(app)
}
func (p *CacheServiceProvider) Boot(app *core.Application) error { return nil }

type EventsServiceProvider struct{}

func (p *EventsServiceProvider) Register(app *core.Application) error {
	return BootEventsServices(app)
}
func (p *EventsServiceProvider) Boot(app *core.Application) error { return nil }

type QueueServiceProvider struct{}

func (p *QueueServiceProvider) Register(app *core.Application) error {
	return BootQueueServices(app)
}
func (p *QueueServiceProvider) Boot(app *core.Application) error { return nil }

type AuthServiceProvider struct{}

func (p *AuthServiceProvider) Register(app *core.Application) error {
	return BootAuthServices(app)
}
func (p *AuthServiceProvider) Boot(app *core.Application) error { return nil }

type LocalizationServiceProvider struct{}

func (p *LocalizationServiceProvider) Register(app *core.Application) error {
	return BootLocalizationServices(app)
}
func (p *LocalizationServiceProvider) Boot(app *core.Application) error { return nil }

type ScheduleServiceProvider struct{}

func (p *ScheduleServiceProvider) Register(app *core.Application) error {
	return BootScheduleServices(app)
}
func (p *ScheduleServiceProvider) Boot(app *core.Application) error { return nil }

type HTTPClientServiceProvider struct{}

func (p *HTTPClientServiceProvider) Register(app *core.Application) error {
	return BootHTTPClientServices(app)
}
func (p *HTTPClientServiceProvider) Boot(app *core.Application) error { return nil }

type BroadcastingServiceProvider struct{}

func (p *BroadcastingServiceProvider) Register(app *core.Application) error {
	return BootBroadcastingServices(app)
}
func (p *BroadcastingServiceProvider) Boot(app *core.Application) error { return nil }

type NotificationServiceProvider struct{}

func (p *NotificationServiceProvider) Register(app *core.Application) error {
	return BootNotificationServices(app)
}
func (p *NotificationServiceProvider) Boot(app *core.Application) error { return nil }

type FilesystemServiceProvider struct{}

func (p *FilesystemServiceProvider) Register(app *core.Application) error {
	return BootFilesystemServices(app)
}
func (p *FilesystemServiceProvider) Boot(app *core.Application) error { return nil }

type AssetsServiceProvider struct{}

func (p *AssetsServiceProvider) Register(app *core.Application) error {
	return BootAssetsServices(app)
}
func (p *AssetsServiceProvider) Boot(app *core.Application) error { return nil }

type ViewSessionServiceProvider struct{}

func (p *ViewSessionServiceProvider) Register(app *core.Application) error {
	return BootViewSession(app)
}
func (p *ViewSessionServiceProvider) Boot(app *core.Application) error { return nil }
