package foundation

import "github.com/zatrano/framework/kernel"

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

func (p *KernelServiceProvider) Register(app *kernel.Application) error {
	return app.BootKernelServices()
}
func (p *KernelServiceProvider) Boot(app *kernel.Application) error { return nil }

type DatabaseServiceProvider struct{}

func (p *DatabaseServiceProvider) Register(app *kernel.Application) error {
	return BootDatabase(app)
}
func (p *DatabaseServiceProvider) Boot(app *kernel.Application) error { return nil }

type CacheServiceProvider struct{}

func (p *CacheServiceProvider) Register(app *kernel.Application) error {
	return BootCacheServices(app)
}
func (p *CacheServiceProvider) Boot(app *kernel.Application) error { return nil }

type EventsServiceProvider struct{}

func (p *EventsServiceProvider) Register(app *kernel.Application) error {
	return BootEventsServices(app)
}
func (p *EventsServiceProvider) Boot(app *kernel.Application) error { return nil }

type QueueServiceProvider struct{}

func (p *QueueServiceProvider) Register(app *kernel.Application) error {
	return BootQueueServices(app)
}
func (p *QueueServiceProvider) Boot(app *kernel.Application) error { return nil }

type AuthServiceProvider struct{}

func (p *AuthServiceProvider) Register(app *kernel.Application) error {
	return BootAuthServices(app)
}
func (p *AuthServiceProvider) Boot(app *kernel.Application) error { return nil }

type LocalizationServiceProvider struct{}

func (p *LocalizationServiceProvider) Register(app *kernel.Application) error {
	return BootLocalizationServices(app)
}
func (p *LocalizationServiceProvider) Boot(app *kernel.Application) error { return nil }

type ScheduleServiceProvider struct{}

func (p *ScheduleServiceProvider) Register(app *kernel.Application) error {
	return BootScheduleServices(app)
}
func (p *ScheduleServiceProvider) Boot(app *kernel.Application) error { return nil }

type HTTPClientServiceProvider struct{}

func (p *HTTPClientServiceProvider) Register(app *kernel.Application) error {
	return BootHTTPClientServices(app)
}
func (p *HTTPClientServiceProvider) Boot(app *kernel.Application) error { return nil }

type BroadcastingServiceProvider struct{}

func (p *BroadcastingServiceProvider) Register(app *kernel.Application) error {
	return BootBroadcastingServices(app)
}
func (p *BroadcastingServiceProvider) Boot(app *kernel.Application) error { return nil }

type NotificationServiceProvider struct{}

func (p *NotificationServiceProvider) Register(app *kernel.Application) error {
	return BootNotificationServices(app)
}
func (p *NotificationServiceProvider) Boot(app *kernel.Application) error { return nil }

type FilesystemServiceProvider struct{}

func (p *FilesystemServiceProvider) Register(app *kernel.Application) error {
	return BootFilesystemServices(app)
}
func (p *FilesystemServiceProvider) Boot(app *kernel.Application) error { return nil }

type AssetsServiceProvider struct{}

func (p *AssetsServiceProvider) Register(app *kernel.Application) error {
	return BootAssetsServices(app)
}
func (p *AssetsServiceProvider) Boot(app *kernel.Application) error { return nil }

type ViewSessionServiceProvider struct{}

func (p *ViewSessionServiceProvider) Register(app *kernel.Application) error {
	return BootViewSession(app)
}
func (p *ViewSessionServiceProvider) Boot(app *kernel.Application) error { return nil }
