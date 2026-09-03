package bootstrap

// Side-effect imports so addon packages register themselves with bootstrap/addons.
// Consumer apps should blank-import only the addons they enable; this repo imports
// the full service set so package:enable / App(WithDemo()) / tests keep working.
import (
	_ "github.com/zatrano/framework/packages/ai"
	_ "github.com/zatrano/framework/packages/audit"
	_ "github.com/zatrano/framework/packages/backup"
	_ "github.com/zatrano/framework/packages/billing"
	_ "github.com/zatrano/framework/packages/bus"
	_ "github.com/zatrano/framework/packages/circuit"
	_ "github.com/zatrano/framework/packages/docs"
	_ "github.com/zatrano/framework/packages/enums"
	_ "github.com/zatrano/framework/packages/features"
	_ "github.com/zatrano/framework/packages/geo"
	_ "github.com/zatrano/framework/packages/graphql"
	_ "github.com/zatrano/framework/packages/hashid"
	_ "github.com/zatrano/framework/packages/inspector"
	_ "github.com/zatrano/framework/packages/lock"
	_ "github.com/zatrano/framework/packages/oauth"
	_ "github.com/zatrano/framework/packages/octane"
	_ "github.com/zatrano/framework/packages/otp"
	_ "github.com/zatrano/framework/packages/pulse"
	_ "github.com/zatrano/framework/packages/search"
	_ "github.com/zatrano/framework/packages/shorturl"
	_ "github.com/zatrano/framework/packages/sitemap"
	_ "github.com/zatrano/framework/packages/social"
	_ "github.com/zatrano/framework/packages/tenancy"
	_ "github.com/zatrano/framework/packages/webhooks"
	_ "github.com/zatrano/framework/packages/wellknown"
)
