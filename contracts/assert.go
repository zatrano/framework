package contracts

import (
	"github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/container"
	appcontext "github.com/zatrano/framework/packages/context"
	"github.com/zatrano/framework/packages/encryption"
	"github.com/zatrano/framework/packages/exceptions"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/health"
	"github.com/zatrano/framework/packages/log"
	"github.com/zatrano/framework/packages/maintenance"
	"github.com/zatrano/framework/packages/observability"
	"github.com/zatrano/framework/packages/ratelimit"
	"github.com/zatrano/framework/packages/report"
	"github.com/zatrano/framework/packages/routing"
	urlgen "github.com/zatrano/framework/packages/url"
)

// Compile-time assertions: kernel concrete types satisfy the SemVer contracts.
var (
	_ Container        = (*container.Container)(nil)
	_ ConfigRepository = (*config.Repository)(nil)
	_ Router           = (*routing.Router)(nil)
	_ Logger           = (*log.Logger)(nil)
	_ RateLimiter      = (*ratelimit.Limiter)(nil)
	_ ContextStore     = (*appcontext.Store)(nil)
	_ URLGenerator     = (*urlgen.Generator)(nil)
	_ Encrypter        = (*encryption.Encrypter)(nil)
	_ Hasher           = (*hashing.Manager)(nil)
	_ Metrics          = (*observability.Metrics)(nil)
	_ Health           = (*health.Manager)(nil)
	_ Maintenance      = (*maintenance.Manager)(nil)
	_ Exceptions       = (*exceptions.Handler)(nil)
	_ Reports          = (*report.Manager)(nil)
)
