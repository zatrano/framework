package bootstrap

// DemoAddons is the full first-party addon set for App(WithDemo()) / local exploration.
// Production apps should keep EnabledAddons explicit and minimal instead.
var DemoAddons = []string{
	"ai",
	"audit",
	"backup",
	"billing",
	"bus",
	"circuit",
	"docs",
	"enums",
	"features",
	"geo",
	"graphql",
	"hashid",
	"inspector",
	"lock",
	"mongo",
	"oauth",
	"octane",
	"otp",
	"pulse",
	"search",
	"shorturl",
	"sitemap",
	"social",
	"tenancy",
	"webauthn",
	"webhooks",
	"wellknown",
}
