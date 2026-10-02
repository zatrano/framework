package ssr

// Engine is the framework-facing SSR contract (default: Canvas).
type Engine interface {
	Render(name string, data map[string]any) (string, error)
	Component(name string, data map[string]any) (string, error)
	Share(key string, value any)
	AddFunc(name string, fn any)
	EnableCache(enabled bool)
	SetEnvironment(env string)
}

// Factory builds an Engine rooted at directory.
type Factory func(directory string) Engine

var defaultFactory Factory = NewCanvas

// SetFactory replaces the default engine factory (nil restores Canvas).
func SetFactory(f Factory) {
	if f == nil {
		defaultFactory = NewCanvas
		return
	}
	defaultFactory = f
}

// New creates an Engine using the configured factory (Canvas by default).
func New(directory string) Engine {
	return defaultFactory(directory)
}
