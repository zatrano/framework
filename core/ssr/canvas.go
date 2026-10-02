package ssr

import "github.com/zatrano/canvas"

// NewCanvas builds the default Canvas-backed Engine.
func NewCanvas(directory string) Engine {
	return &canvasEngine{inner: canvas.New(directory)}
}

type canvasEngine struct {
	inner *canvas.Engine
}

func (c *canvasEngine) Render(name string, data map[string]any) (string, error) {
	return c.inner.Render(name, data)
}

func (c *canvasEngine) Component(name string, data map[string]any) (string, error) {
	return c.inner.Component(name, data)
}

func (c *canvasEngine) Share(key string, value any) {
	c.inner.Share(key, value)
}

func (c *canvasEngine) AddFunc(name string, fn any) {
	c.inner.AddFunc(name, fn)
}

func (c *canvasEngine) EnableCache(enabled bool) {
	c.inner.EnableCache(enabled)
}

func (c *canvasEngine) SetEnvironment(env string) {
	c.inner.SetEnvironment(env)
}

// Canvas returns the underlying Canvas engine when present.
func Canvas(e Engine) (*canvas.Engine, bool) {
	c, ok := e.(*canvasEngine)
	if !ok || c == nil {
		return nil, false
	}
	return c.inner, true
}
