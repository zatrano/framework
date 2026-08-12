package tenancy

import (
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const (
	AttrTenant   = "tenant"
	AttrTenantID = "tenant_id"
	HeaderTenant = "X-Tenant"
)

// Tenant represents the current tenant.
type Tenant struct {
	ID     string         `json:"id"`
	Name   string         `json:"name,omitempty"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

// Resolver resolves a tenant from the request.
type Resolver func(req *http.Request) (*Tenant, error)

// Manager stores tenants and resolves the current one.
type Manager struct {
	mu         sync.RWMutex
	tenants    map[string]*Tenant
	byDomain   map[string]string
	resolver   Resolver
	bootstraps []func(t *Tenant) error
}

// New creates a tenancy manager.
func New() *Manager {
	return &Manager{
		tenants:  map[string]*Tenant{},
		byDomain: map[string]string{},
		resolver: defaultResolver,
	}
}

// Register adds a tenant.
func (m *Manager) Register(tenant Tenant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := tenant
	if copy.Data == nil {
		copy.Data = map[string]any{}
	}
	m.tenants[copy.ID] = &copy
	if copy.Domain != "" {
		m.byDomain[strings.ToLower(copy.Domain)] = copy.ID
	}
}

// Find returns a tenant by id.
func (m *Manager) Find(id string) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tenants[id]
	return t, ok
}

// FindByDomain returns a tenant by domain.
func (m *Manager) FindByDomain(domain string) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byDomain[strings.ToLower(domain)]
	if !ok {
		return nil, false
	}
	t, ok := m.tenants[id]
	return t, ok
}

// SetResolver overrides tenant resolution.
func (m *Manager) SetResolver(resolver Resolver) {
	m.resolver = resolver
}

// Bootstrapping registers a hook invoked when a tenant is resolved.
func (m *Manager) Bootstrapping(fn func(t *Tenant) error) {
	m.bootstraps = append(m.bootstraps, fn)
}

// Resolve resolves and attaches the tenant to the request.
func (m *Manager) Resolve(req *http.Request) (*Tenant, error) {
	resolver := m.resolver
	if resolver == nil {
		resolver = defaultResolver
	}
	tenant, err := resolver(req)
	if err != nil || tenant == nil {
		return tenant, err
	}
	for _, hook := range m.bootstraps {
		if err := hook(tenant); err != nil {
			return nil, err
		}
	}
	req.Set(AttrTenant, tenant)
	req.Set(AttrTenantID, tenant.ID)
	return tenant, nil
}

// Current returns the tenant stored on the request.
func Current(req *http.Request) *Tenant {
	if value, ok := req.Get(AttrTenant).(*Tenant); ok {
		return value
	}
	return nil
}

// ID returns the current tenant id.
func ID(req *http.Request) string {
	if value, ok := req.Get(AttrTenantID).(string); ok {
		return value
	}
	if t := Current(req); t != nil {
		return t.ID
	}
	return ""
}

// Middleware resolves the tenant for each request.
func (m *Manager) Middleware(required bool) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			tenant, err := m.Resolve(req)
			if err != nil {
				return http.JSON(map[string]any{"message": err.Error()}).Status(500)
			}
			if required && tenant == nil {
				return http.JSON(map[string]any{"message": "Tenant required"}).Status(400)
			}
			resp := next(req)
			if resp != nil && tenant != nil {
				resp.Header(HeaderTenant, tenant.ID)
			}
			return resp
		}
	}
}

func defaultResolver(req *http.Request) (*Tenant, error) {
	// Placeholder; Manager.Resolve wraps this with registry lookups via SetResolver.
	return nil, nil
}

// HeaderOrHostResolver resolves tenant from X-Tenant header or Host subdomain/domain.
func (m *Manager) HeaderOrHostResolver() Resolver {
	return func(req *http.Request) (*Tenant, error) {
		if id := req.Header(HeaderTenant); id != "" {
			if t, ok := m.Find(id); ok {
				return t, nil
			}
			return &Tenant{ID: id, Name: id}, nil
		}
		if id := req.Query("tenant"); id != "" {
			if t, ok := m.Find(id); ok {
				return t, nil
			}
			return &Tenant{ID: id, Name: id}, nil
		}
		host := req.Header("Host")
		if host == "" && req.Raw() != nil {
			host = req.Raw().Host
		}
		host = strings.Split(host, ":")[0]
		if t, ok := m.FindByDomain(host); ok {
			return t, nil
		}
		// subdomain.example.test -> subdomain
		parts := strings.Split(host, ".")
		if len(parts) > 2 {
			sub := parts[0]
			if sub != "www" {
				if t, ok := m.Find(sub); ok {
					return t, nil
				}
			}
		}
		return nil, nil
	}
}
