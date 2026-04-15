package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Request generates form request struct stubs under baseDir/<name>/requests/.
// Creates create_<name>.go and update_<name>.go with validation tags.
func Request(moduleRoot, baseDir, rawName string, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid request name %q (use letters, digits, _ or -)", rawName)
	}
	pascal := snakeToPascal(name)
	reqDir := filepath.Join(baseDir, name, "requests")

	files := map[string]string{
		"create_" + name + ".go": tmplCreateRequest(name, pascal),
		"update_" + name + ".go": tmplUpdateRequest(name, pascal),
	}

	var written []string
	for fn, body := range files {
		path := filepath.Join(reqDir, fn)
		written = append(written, path)
		if dryRun {
			continue
		}
		if err := os.MkdirAll(reqDir, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, err
		}
	}
	return written, nil
}

func tmplCreateRequest(pkg, pascal string) string {
	return fmt.Sprintf(`package requests

// Create%[2]sRequest is the form request for creating a %[2]s.
// Add your fields and validation tags below.
//
// Struct tags use go-playground/validator syntax.
// See: https://pkg.go.dev/github.com/go-playground/validator/v10
//
// Usage in handlers:
//
//	req, err := zatrano.Validate[requests.Create%[2]sRequest](c)
//	if err != nil { return err }
type Create%[2]sRequest struct {
	Name  string `+"`"+`json:"name"  validate:"required,min=2,max=255"`+"`"+`
	Email string `+"`"+`json:"email" validate:"required,email"`+"`"+`
}
`, pkg, pascal)
}

func tmplUpdateRequest(pkg, pascal string) string {
	return fmt.Sprintf(`package requests

// Update%[2]sRequest is the form request for updating a %[2]s.
// Add your fields and validation tags below.
//
// Usage in handlers:
//
//	req, err := zatrano.Validate[requests.Update%[2]sRequest](c)
//	if err != nil { return err }
type Update%[2]sRequest struct {
	Name  string `+"`"+`json:"name"  validate:"omitempty,min=2,max=255"`+"`"+`
	Email string `+"`"+`json:"email" validate:"omitempty,email"`+"`"+`
}
`, pkg, pascal)
}
