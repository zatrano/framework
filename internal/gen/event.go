package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Event generates an event struct file.
// Creates: baseDir/events/<name>_event.go
func Event(moduleRoot, baseDir, rawName string, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid event name %q (use letters, digits, _ or -)", rawName)
	}
	modPath, err := ModuleImportPath(moduleRoot)
	if err != nil {
		return nil, err
	}
	pascal := snakeToPascal(name)

	dir := filepath.Join(baseDir, "events")
	file := name + "_event.go"
	body := tmplEvent(name, pascal, modPath)

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

func tmplEvent(pkg, pascal, modImport string) string {
	return fmt.Sprintf(`package events

import (
	"github.com/zatrano/framework/pkg/events"
)

// %[2]sEvent is fired when %[1]s occurs.
//
// Fire it with:
//
//	app.Events.Fire(ctx, &myevents.%[2]sEvent{
//	    // set fields...
//	})
type %[2]sEvent struct {
	events.BaseEvent
	// Add event-specific fields here.
}

// Name returns the event identifier.
func (e *%[2]sEvent) Name() string { return "%[1]s" }
`, pkg, pascal, modImport)
}
