package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Listener generates a listener struct file.
// Creates: baseDir/listeners/<name>_listener.go
func Listener(moduleRoot, baseDir, rawName string, queued, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid listener name %q (use letters, digits, _ or -)", rawName)
	}
	modPath, err := ModuleImportPath(moduleRoot)
	if err != nil {
		return nil, err
	}
	pascal := snakeToPascal(name)

	dir := filepath.Join(baseDir, "listeners")
	file := name + "_listener.go"
	var body string
	if queued {
		body = tmplQueuedListener(name, pascal, modPath)
	} else {
		body = tmplListener(name, pascal, modPath)
	}

	outPath := filepath.Join(dir, file)
	if dryRun {
		return []string{outPath}, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(outPath, []byte(body), 0o644); err != nil {
		return nil, err
	}
	return []string{outPath}, nil
}

func tmplListener(pkg, pascal, modImport string) string {
	return fmt.Sprintf(`package listeners

import (
	"context"

	"github.com/zatrano/framework/pkg/events"
)

// %[2]sListener handles the %[1]s event synchronously.
//
// Register it with:
//
//	app.Events.Listen("%[1]s", &listeners.%[2]sListener{})
type %[2]sListener struct{}

// Handle processes the event.
func (l *%[2]sListener) Handle(ctx context.Context, event events.Event) error {
	// TODO: implement %[1]s event handling
	return nil
}
`, pkg, pascal, modImport)
}

func tmplQueuedListener(pkg, pascal, modImport string) string {
	return fmt.Sprintf(`package listeners

import (
	"context"

	"github.com/zatrano/framework/pkg/events"
)

// %[2]sListener handles the %[1]s event asynchronously via the queue.
//
// Register it with:
//
//	app.Events.Listen("%[1]s", &listeners.%[2]sListener{})
type %[2]sListener struct{}

// Handle processes the event.
func (l *%[2]sListener) Handle(ctx context.Context, event events.Event) error {
	// TODO: implement %[1]s event handling
	return nil
}

// Queue returns the queue name for async processing.
func (l *%[2]sListener) Queue() string { return "events" }

// Retries returns the maximum retry attempts.
func (l *%[2]sListener) Retries() int { return 3 }
`, pkg, pascal, modImport)
}
