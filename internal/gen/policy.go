package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Policy generates a policy stub under baseDir/<name>/policies/<name>_policy.go.
// The policy implements the auth.Policy interface with CRUD methods.
func Policy(moduleRoot, baseDir, rawName string, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid policy name %q (use letters, digits, _ or -)", rawName)
	}
	modPath, err := ModuleImportPath(moduleRoot)
	if err != nil {
		return nil, err
	}
	pascal := snakeToPascal(name)
	policyDir := filepath.Join(baseDir, name, "policies")
	fileName := name + "_policy.go"
	body := tmplPolicy(name, pascal, modPath)

	path := filepath.Join(policyDir, fileName)
	var written []string
	written = append(written, path)

	if dryRun {
		return written, nil
	}
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return nil, err
	}
	return written, nil
}

func tmplPolicy(pkg, pascal, modImport string) string {
	return fmt.Sprintf(`package policies

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/auth"
	"github.com/zatrano/framework/pkg/middleware"
)

// %[2]sPolicy defines authorization rules for the %[2]s resource.
// Implements auth.Policy — register it with:
//
//	gate.RegisterPolicy("%[1]s", &policies.%[2]sPolicy{})
//
// Each method receives the Fiber context (use middleware.LocalsUserID / LocalsUserRoles
// to get the authenticated user) and the resource instance where applicable.
type %[2]sPolicy struct{}

// Ensure %[2]sPolicy implements auth.Policy at compile time.
var _ auth.Policy = (*%[2]sPolicy)(nil)

// ViewAny determines if the user can list %[1]s resources.
func (p *%[2]sPolicy) ViewAny(c fiber.Ctx) bool {
	// Example: allow any authenticated user.
	_, ok := c.Locals(middleware.LocalsUserID).(uint)
	return ok
}

// View determines if the user can view a specific %[1]s.
func (p *%[2]sPolicy) View(c fiber.Ctx, resource any) bool {
	// TODO: check ownership or role-based access.
	// Example:
	//   item := resource.(*models.%[2]s)
	//   userID, _ := c.Locals(middleware.LocalsUserID).(uint)
	//   return item.UserID == userID
	return true
}

// Create determines if the user can create a new %[1]s.
func (p *%[2]sPolicy) Create(c fiber.Ctx) bool {
	_, ok := c.Locals(middleware.LocalsUserID).(uint)
	return ok
}

// Update determines if the user can update a specific %[1]s.
func (p *%[2]sPolicy) Update(c fiber.Ctx, resource any) bool {
	// TODO: check ownership or permissions.
	return true
}

// Delete determines if the user can delete a specific %[1]s.
func (p *%[2]sPolicy) Delete(c fiber.Ctx, resource any) bool {
	// TODO: check ownership or permissions.
	return true
}

// ForceDelete determines if the user can permanently delete a %[1]s (soft-delete scenarios).
func (p *%[2]sPolicy) ForceDelete(c fiber.Ctx, resource any) bool {
	// Typically restricted to admin roles.
	return false
}

// Restore determines if the user can restore a soft-deleted %[1]s.
func (p *%[2]sPolicy) Restore(c fiber.Ctx, resource any) bool {
	// Typically restricted to admin roles.
	return false
}
`, pkg, pascal, modImport)
}
