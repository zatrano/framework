package authorization

import (
	"fmt"
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Authenticatable is the minimal user contract for gates (satisfied by auth users).
type Authenticatable interface {
	AuthID() any
	AuthPassword() string
}

// GateFunc authorizes an ability for a user against optional arguments.
type GateFunc func(user Authenticatable, arguments ...any) bool

// BeforeFunc may short-circuit authorization. Non-nil return wins.
type BeforeFunc func(user Authenticatable, ability string, arguments ...any) *bool

// AfterFunc may override the authorization result. Non-nil return wins.
type AfterFunc func(user Authenticatable, ability string, result bool, arguments ...any) *bool

// Policy maps abilities to authorization callbacks for a resource type.
type Policy struct {
	abilities map[string]GateFunc
}

// NewPolicy creates an empty policy.
func NewPolicy() *Policy {
	return &Policy{abilities: make(map[string]GateFunc)}
}

// Define registers an ability on the policy.
func (p *Policy) Define(ability string, fn GateFunc) *Policy {
	p.abilities[ability] = fn
	return p
}

// Has reports whether the policy defines an ability.
func (p *Policy) Has(ability string) bool {
	_, ok := p.abilities[ability]
	return ok
}

// Gate is the authorization manager.
type Gate struct {
	mu              sync.RWMutex
	abilities       map[string]GateFunc
	policies        map[string]*Policy
	beforeCallbacks []BeforeFunc
	afterCallbacks  []AfterFunc
}

// New creates a gate.
func New() *Gate {
	return &Gate{
		abilities: make(map[string]GateFunc),
		policies:  make(map[string]*Policy),
	}
}

// Define registers a gate ability.
func (g *Gate) Define(ability string, fn GateFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.abilities[ability] = fn
}

// Policy registers a named policy.
func (g *Gate) Policy(name string, policy *Policy) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.policies[name] = policy
}

// Has reports whether an ability or policy ability is defined.
func (g *Gate) Has(ability string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if _, ok := g.abilities[ability]; ok {
		return true
	}
	if policyName, policyAbility, ok := splitAbility(ability); ok {
		if policy := g.policies[policyName]; policy != nil {
			return policy.Has(policyAbility)
		}
	}
	return false
}

// Before registers a global before callback.
// Return non-nil to short-circuit authorization.
func (g *Gate) Before(fn BeforeFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.beforeCallbacks = append(g.beforeCallbacks, fn)
}

// After registers a global after callback.
// Return non-nil to override the authorization result.
func (g *Gate) After(fn AfterFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.afterCallbacks = append(g.afterCallbacks, fn)
}

// ForUser returns a user-scoped authorization helper.
func (g *Gate) ForUser(user Authenticatable) *PendingGate {
	return &PendingGate{gate: g, user: user}
}

// Allows reports whether the user may perform the ability.
func (g *Gate) Allows(user Authenticatable, ability string, arguments ...any) bool {
	return g.Inspect(user, ability, arguments...).Allowed()
}

// Check is an alias for Allows.
func (g *Gate) Check(user Authenticatable, ability string, arguments ...any) bool {
	return g.Allows(user, ability, arguments...)
}

// Denies is the inverse of Allows.
func (g *Gate) Denies(user Authenticatable, ability string, arguments ...any) bool {
	return !g.Allows(user, ability, arguments...)
}

// Any reports whether the user is allowed any of the abilities.
func (g *Gate) Any(user Authenticatable, abilities []string, arguments ...any) bool {
	for _, ability := range abilities {
		if g.Allows(user, ability, arguments...) {
			return true
		}
	}
	return false
}

// None reports whether the user is denied all of the abilities.
func (g *Gate) None(user Authenticatable, abilities []string, arguments ...any) bool {
	return !g.Any(user, abilities, arguments...)
}

// Response is an authorization response.
type Response struct {
	allowed bool
	message string
}

// Allowed reports whether authorization succeeded.
func (r Response) Allowed() bool { return r.allowed }

// Message returns an optional denial message.
func (r Response) Message() string { return r.message }

// Allow creates an allow response.
func Allow(message ...string) Response {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	return Response{allowed: true, message: msg}
}

// Deny creates a deny response.
func Deny(message ...string) Response {
	msg := "This action is unauthorized."
	if len(message) > 0 {
		msg = message[0]
	}
	return Response{allowed: false, message: msg}
}

// AuthorizationException is returned by Authorize when access is denied.
type AuthorizationException struct {
	Ability string
	Message string
}

func (e AuthorizationException) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "This action is unauthorized."
}

// ResponseFor converts an authorization failure into an HTTP response.
func ResponseFor(err error) *http.Response {
	switch e := err.(type) {
	case AuthorizationException:
		return http.JSON(map[string]any{
			"message": e.Error(),
			"ability": e.Ability,
		}).Status(403)
	default:
		if err == nil {
			return nil
		}
		return http.JSON(map[string]any{"message": err.Error()}).Status(403)
	}
}

// Inspect evaluates an ability and returns a response.
func (g *Gate) Inspect(user Authenticatable, ability string, arguments ...any) Response {
	g.mu.RLock()
	befores := append([]BeforeFunc{}, g.beforeCallbacks...)
	afters := append([]AfterFunc{}, g.afterCallbacks...)
	abilityFn := g.abilities[ability]
	g.mu.RUnlock()

	var resp Response
	shortCircuited := false

	for _, before := range befores {
		if result := before(user, ability, arguments...); result != nil {
			shortCircuited = true
			if *result {
				resp = Allow()
			} else {
				resp = Deny()
			}
			break
		}
	}

	if !shortCircuited {
		if abilityFn != nil {
			if abilityFn(user, arguments...) {
				resp = Allow()
			} else {
				resp = Deny()
			}
		} else if policyName, policyAbility, ok := splitAbility(ability); ok {
			g.mu.RLock()
			policy := g.policies[policyName]
			g.mu.RUnlock()
			if policy != nil {
				if fn, exists := policy.abilities[policyAbility]; exists {
					if fn(user, arguments...) {
						resp = Allow()
					} else {
						resp = Deny()
					}
				} else {
					resp = Deny(fmt.Sprintf("ability [%s] is not defined", ability))
				}
			} else {
				resp = Deny(fmt.Sprintf("ability [%s] is not defined", ability))
			}
		} else {
			resp = Deny(fmt.Sprintf("ability [%s] is not defined", ability))
		}
	}

	for _, after := range afters {
		if override := after(user, ability, resp.Allowed(), arguments...); override != nil {
			if *override {
				return Allow()
			}
			return Deny()
		}
	}
	return resp
}

// Authorize returns an AuthorizationException when denied.
func (g *Gate) Authorize(user Authenticatable, ability string, arguments ...any) error {
	resp := g.Inspect(user, ability, arguments...)
	if resp.Allowed() {
		return nil
	}
	return AuthorizationException{Ability: ability, Message: resp.Message()}
}

// PendingGate is a user-scoped gate helper returned by ForUser.
type PendingGate struct {
	gate *Gate
	user Authenticatable
}

// Allows reports whether the bound user may perform the ability.
func (p *PendingGate) Allows(ability string, arguments ...any) bool {
	return p.gate.Allows(p.user, ability, arguments...)
}

// Check is an alias for Allows.
func (p *PendingGate) Check(ability string, arguments ...any) bool {
	return p.Allows(ability, arguments...)
}

// Denies is the inverse of Allows.
func (p *PendingGate) Denies(ability string, arguments ...any) bool {
	return p.gate.Denies(p.user, ability, arguments...)
}

// Any reports whether the bound user is allowed any of the abilities.
func (p *PendingGate) Any(abilities []string, arguments ...any) bool {
	return p.gate.Any(p.user, abilities, arguments...)
}

// None reports whether the bound user is denied all of the abilities.
func (p *PendingGate) None(abilities []string, arguments ...any) bool {
	return p.gate.None(p.user, abilities, arguments...)
}

// Inspect evaluates an ability for the bound user.
func (p *PendingGate) Inspect(ability string, arguments ...any) Response {
	return p.gate.Inspect(p.user, ability, arguments...)
}

// Authorize authorizes an ability for the bound user.
func (p *PendingGate) Authorize(ability string, arguments ...any) error {
	return p.gate.Authorize(p.user, ability, arguments...)
}

// Middleware authorizes an ability using the authenticated user.
// currentUser should return an Authenticatable (e.g. auth.From(app).User(req)).
// Optional arguments are forwarded to the gate (e.g. route model ids).
func Middleware(gate *Gate, currentUser func(*http.Request) any, ability string, arguments ...any) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			user := asAuthenticatable(currentUser(req))
			if user == nil {
				return http.JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
			}
			args := resolveMiddlewareArgs(req, arguments...)
			if err := gate.Authorize(user, ability, args...); err != nil {
				return ResponseFor(err)
			}
			return next(req)
		}
	}
}

// MiddlewareAny authorizes when the user is allowed any of the abilities.
func MiddlewareAny(gate *Gate, currentUser func(*http.Request) any, abilities []string, arguments ...any) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			user := asAuthenticatable(currentUser(req))
			if user == nil {
				return http.JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
			}
			args := resolveMiddlewareArgs(req, arguments...)
			if !gate.Any(user, abilities, args...) {
				return ResponseFor(AuthorizationException{
					Ability: strings.Join(abilities, "|"),
					Message: "This action is unauthorized.",
				})
			}
			return next(req)
		}
	}
}

func asAuthenticatable(raw any) Authenticatable {
	if raw == nil {
		return nil
	}
	if u, ok := raw.(Authenticatable); ok {
		return u
	}
	return nil
}

func resolveMiddlewareArgs(req *http.Request, arguments ...any) []any {
	if len(arguments) == 0 {
		return nil
	}
	out := make([]any, len(arguments))
	for i, arg := range arguments {
		switch v := arg.(type) {
		case func(*http.Request) any:
			out[i] = v(req)
		default:
			out[i] = arg
		}
	}
	return out
}

func splitAbility(ability string) (string, string, bool) {
	for i := 0; i < len(ability); i++ {
		if ability[i] == '.' {
			return ability[:i], ability[i+1:], true
		}
	}
	return "", "", false
}
