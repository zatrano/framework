package routes

import "github.com/zatrano/framework/kernel"

var application *kernel.Application

// Use binds the application for self-registered route functions.
func Use(app *kernel.Application) {
	application = app
}

func currentApp() *kernel.Application {
	return application
}
