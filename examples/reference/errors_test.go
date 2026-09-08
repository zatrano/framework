package reference

import (
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
	"github.com/zatrano/framework/v2/examples/reference/internal/repository"
	"github.com/zatrano/framework/v2/examples/reference/internal/service"
)

func TestHTTPUnavailableDoesNotLeakCause(t *testing.T) {
	secret := "super-secret-value-xyz"
	app := bootAndStart(t, &swapService{svc: service.New(&repository.Unavailable{
		Cause: errors.New("dsn=postgres://u:" + secret + "@db"),
	})})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "http://example/api/v1/items/1", nil))
	if rec.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), secret) {
		t.Fatalf("secret leaked: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "dependency unavailable") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestConfigurationFailureWraps(t *testing.T) {
	t.Setenv(envPollMS, "nope")
	app := Assemble(t.TempDir())
	t.Cleanup(func() { closeLog(t, app) })
	err := app.Bootstrap()
	if err == nil {
		t.Fatal("expected configuration failure")
	}
	if !errors.Is(err, domain.ErrConfiguration) {
		t.Fatalf("want ErrConfiguration: %v", err)
	}
	if !app.BootstrapFailed() {
		t.Fatal("expected BootFailed")
	}
}

type swapService struct {
	svc *service.Items
}

func (p *swapService) Register(app contracts.App) error {
	app.Container().Instance(keyService, p.svc)
	return nil
}

func (p *swapService) Boot(contracts.App) error { return nil }
