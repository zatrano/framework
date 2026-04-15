package auth

import (
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v3"
)

// ─── Gate — resource-based authorization ────────────────────────────────────
//
// Usage:
//
//	gate := auth.NewGate()
//	gate.Define("edit-post", func(c fiber.Ctx, resource any) bool {
//	    post := resource.(*Post)
//	    userID, _ := c.Locals("user_id").(uint)
//	    return post.AuthorID == userID
//	})
//
//	// In handler:
//	if !gate.Allows(c, "edit-post", post) {
//	    return fiber.NewError(fiber.StatusForbidden, "unauthorized")
//	}

// GateFunc is the callback signature for gate definitions.
// It receives the Fiber context (which holds user info via Locals) and the resource being checked.
type GateFunc func(c fiber.Ctx, resource any) bool

// Gate is a registry of named authorization checks.
type Gate struct {
	mu    sync.RWMutex
	gates map[string]GateFunc

	// before callbacks run before the gate check; if any returns a non-nil *bool,
	// that value is used and the gate callback is skipped. Useful for "super-admin" bypass.
	beforeCallbacks []BeforeFunc
}

// BeforeFunc runs before every gate check. Return *true to always allow, *false to always deny,
// or nil to fall through to the gate definition.
type BeforeFunc func(c fiber.Ctx, ability string, resource any) *bool

// NewGate creates an empty gate registry.
func NewGate() *Gate {
	return &Gate{
		gates: make(map[string]GateFunc),
	}
}

// Define registers an authorization check under the given ability name.
//
//	gate.Define("edit-post", func(c fiber.Ctx, resource any) bool { ... })
func (g *Gate) Define(ability string, fn GateFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.gates[ability] = fn
}

// Before registers a callback that runs before any gate check.
// Useful for super-admin bypass:
//
//	gate.Before(func(c fiber.Ctx, ability string, resource any) *bool {
//	    if isSuperAdmin(c) { t := true; return &t }
//	    return nil
//	})
func (g *Gate) Before(fn BeforeFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.beforeCallbacks = append(g.beforeCallbacks, fn)
}

// Allows returns true if the current user is authorized for the given ability on the resource.
func (g *Gate) Allows(c fiber.Ctx, ability string, resource any) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Run before callbacks first.
	for _, before := range g.beforeCallbacks {
		if result := before(c, ability, resource); result != nil {
			return *result
		}
	}

	fn, ok := g.gates[ability]
	if !ok {
		// Undefined ability → deny by default.
		return false
	}
	return fn(c, resource)
}

// Denies is the inverse of Allows.
func (g *Gate) Denies(c fiber.Ctx, ability string, resource any) bool {
	return !g.Allows(c, ability, resource)
}

// Authorize returns a fiber error (403) if the ability is denied.
// Use in handlers for quick guard checks:
//
//	if err := gate.Authorize(c, "delete-post", post); err != nil {
//	    return err
//	}
func (g *Gate) Authorize(c fiber.Ctx, ability string, resource any) error {
	if g.Denies(c, ability, resource) {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("unauthorized ability: %s", ability))
	}
	return nil
}

// Has returns true if the ability is defined (even if it would deny).
func (g *Gate) Has(ability string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.gates[ability]
	return ok
}

// ─── Policy registration ───────────────────────────────────────────────────

// Policy is the interface for resource-specific authorization policies.
// Each method corresponds to a CRUD action. Implement any subset; unimplemented
// methods should return false.
type Policy interface {
	// ViewAny checks if the user can list the resource.
	ViewAny(c fiber.Ctx) bool
	// View checks if the user can view a specific instance.
	View(c fiber.Ctx, resource any) bool
	// Create checks if the user can create new instances.
	Create(c fiber.Ctx) bool
	// Update checks if the user can update a specific instance.
	Update(c fiber.Ctx, resource any) bool
	// Delete checks if the user can delete a specific instance.
	Delete(c fiber.Ctx, resource any) bool
	// ForceDelete checks if the user can permanently delete (soft-delete scenarios).
	ForceDelete(c fiber.Ctx, resource any) bool
	// Restore checks if the user can restore a soft-deleted instance.
	Restore(c fiber.Ctx, resource any) bool
}

// RegisterPolicy maps CRUD abilities for a resource name to a Policy implementation.
// It creates gate definitions like "<resource>.viewAny", "<resource>.view", etc.
//
//	gate.RegisterPolicy("post", &PostPolicy{})
func (g *Gate) RegisterPolicy(resource string, policy Policy) {
	g.Define(resource+".viewAny", func(c fiber.Ctx, _ any) bool {
		return policy.ViewAny(c)
	})
	g.Define(resource+".view", func(c fiber.Ctx, res any) bool {
		return policy.View(c, res)
	})
	g.Define(resource+".create", func(c fiber.Ctx, _ any) bool {
		return policy.Create(c)
	})
	g.Define(resource+".update", func(c fiber.Ctx, res any) bool {
		return policy.Update(c, res)
	})
	g.Define(resource+".delete", func(c fiber.Ctx, res any) bool {
		return policy.Delete(c, res)
	})
	g.Define(resource+".forceDelete", func(c fiber.Ctx, res any) bool {
		return policy.ForceDelete(c, res)
	})
	g.Define(resource+".restore", func(c fiber.Ctx, res any) bool {
		return policy.Restore(c, res)
	})
}
