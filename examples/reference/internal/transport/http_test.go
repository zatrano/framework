package transport

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	stdhttp "net/http"

	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
	"github.com/zatrano/framework/v2/examples/reference/internal/repository"
	"github.com/zatrano/framework/v2/examples/reference/internal/service"
	"github.com/zatrano/framework/v2/kernel/http"
)

func TestMapErrorDoesNotExposeCause(t *testing.T) {
	secret := "super-secret-value-xyz"
	err := wrap(domain.ErrUnavailable, errors.New("dsn=postgres://user:"+secret+"@db"))
	resp := MapError(err)
	body := string(resp.Content())
	if strings.Contains(body, secret) {
		t.Fatalf("secret leaked: %s", body)
	}
	if !strings.Contains(body, "dependency unavailable") {
		t.Fatalf("body=%s", body)
	}
	if resp.StatusCode() != 503 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestMapErrorDistinguishesSentinels(t *testing.T) {
	cases := []struct {
		err    error
		status int
		msg    string
	}{
		{domain.ErrNotFound, 404, "item not found"},
		{domain.ErrInvalidRequest, 400, "invalid request"},
		{domain.ErrConfiguration, 500, "configuration failure"},
		{errors.New("mystery"), 500, "internal error"},
	}
	for _, c := range cases {
		resp := MapError(c.err)
		if resp.StatusCode() != c.status {
			t.Fatalf("%v status=%d want %d", c.err, resp.StatusCode(), c.status)
		}
		body := string(resp.Content())
		if !strings.Contains(body, c.msg) {
			t.Fatalf("%v body=%s", c.err, body)
		}
	}
}

func TestShowItemNotFound(t *testing.T) {
	h := ShowItem(service.New(repository.NewMemory()))
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/api/v1/items/missing", nil))
	req.SetRouteParams(map[string]string{"id": "missing"})
	resp := h(req)
	if resp.StatusCode() != 404 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func wrap(sentinel, cause error) error {
	return &join{sentinel: sentinel, cause: cause}
}

type join struct {
	sentinel error
	cause    error
}

func (e *join) Error() string        { return e.sentinel.Error() + ": " + e.cause.Error() }
func (e *join) Unwrap() error        { return e.cause }
func (e *join) Is(target error) bool { return target == e.sentinel }
