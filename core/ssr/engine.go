package ssr

import "github.com/zatrano/canvas"

// Factory builds a Canvas engine rooted at directory.
type Factory func(directory string) *canvas.Engine

var defaultFactory Factory = canvas.New

// SetFactory replaces the default engine factory (nil restores canvas.New).
func SetFactory(f Factory) {
	if f == nil {
		defaultFactory = canvas.New
		return
	}
	defaultFactory = f
}

// New creates a Canvas engine using the configured factory.
func New(directory string) *canvas.Engine {
	return defaultFactory(directory)
}
