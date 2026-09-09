package dirs

// CanonicalConsumerDirs are platform layout directories required of every
// application created by zatrano new (empty, web, api, and full profiles).
// Doctor and agents use this list; they must not infer layout from a web template.
func CanonicalConsumerDirs() []string {
	return []string{
		"app/console",
		"app/http/controllers/api",
		"app/http/controllers/web",
		"app/providers",
		"app/routes",
		"app/routes/api",
		"app/routes/web",
		"bootstrap",
		"cmd/app",
	}
}

// CanonicalRouteDirs are where RegisterWeb / RegisterAPI belong.
func CanonicalRouteDirs() []string {
	return []string{
		"app/routes/web",
		"app/routes/api",
	}
}

// OptionalWebScaffoldDirs are web-scaffold / view-package layout, not kernel requirements.
func OptionalWebScaffoldDirs() []string {
	return []string{
		"app/views",
		"app/localization",
		"public/css",
		"public/js",
	}
}
