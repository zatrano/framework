package tenancy_test

import (
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/tenancy"
)

func TestTenantResolver(t *testing.T) {
	m := tenancy.New()
	m.Register(tenancy.Tenant{ID: "acme", Name: "Acme", Domain: "acme.test"})
	m.SetResolver(m.HeaderOrHostResolver())

	raw := httptest.NewRequest("GET", "/api/tenant", nil)
	raw.Header.Set("X-Tenant", "acme")
	req := http.NewRequest(raw)
	tenant, err := m.Resolve(req)
	if err != nil || tenant == nil || tenant.ID != "acme" {
		t.Fatalf("expected acme, got %#v err=%v", tenant, err)
	}
	if tenancy.ID(req) != "acme" {
		t.Fatal("expected tenant id on request")
	}
}
