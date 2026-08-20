package schedule

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/cron"
)

// Event is a scheduled callback.
type Event struct {
	name               string
	expression         string
	callback           func() error
	timezone           *time.Location
	withoutOverlapping bool
	overlapExpires     time.Duration
	outputPath         string
	mutexPath          string
}

// Scheduler stores and runs scheduled events.
type Scheduler struct {
	mu       sync.Mutex
	events   []*Event
	mutexDir string
}

// New creates a scheduler.
func New() *Scheduler {
	return &Scheduler{events: make([]*Event, 0)}
}

// SetMutexPath configures the directory used for overlap locks.
func (s *Scheduler) SetMutexPath(path string) {
	s.mutexDir = path
	_ = os.MkdirAll(path, 0o700)
	_ = os.Chmod(path, 0o700)
}

// Call registers a callback with a cron expression or alias.
func (s *Scheduler) Call(callback func() error) *Event {
	event := &Event{
		callback:   callback,
		expression: "* * * * *",
		timezone:   time.Local,
		mutexPath:  s.mutexDir,
	}
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	return event
}

// Command registers a named scheduled callback.
func (s *Scheduler) Command(name string, callback func() error) *Event {
	return s.Call(callback).Name(name)
}

// Name sets the event name.
func (e *Event) Name(name string) *Event {
	e.name = name
	return e
}

// DisplayName returns the event name or expression.
func (e *Event) DisplayName() string {
	if e.name != "" {
		return e.name
	}
	return e.expression
}

// Expression returns the cron expression.
func (e *Event) Expression() string {
	return e.expression
}

// Timezone sets the event timezone (IANA name or Local/UTC).
func (e *Event) Timezone(name string) *Event {
	if stringsEqualFold(name, "local") {
		e.timezone = time.Local
		return e
	}
	if stringsEqualFold(name, "utc") {
		e.timezone = time.UTC
		return e
	}
	loc, err := time.LoadLocation(name)
	if err == nil {
		e.timezone = loc
	}
	return e
}

// WithoutOverlapping prevents concurrent runs via a lock file.
func (e *Event) WithoutOverlapping(expires ...time.Duration) *Event {
	e.withoutOverlapping = true
	e.overlapExpires = time.Hour
	if len(expires) > 0 && expires[0] > 0 {
		e.overlapExpires = expires[0]
	}
	return e
}

// AppendOutputTo writes callback logs to a file.
func (e *Event) AppendOutputTo(path string) *Event {
	e.outputPath = path
	return e
}

// Cron sets a raw cron expression (min hour dom month dow).
func (e *Event) Cron(expression string) *Event {
	e.expression = expression
	return e
}

// EveryMinute schedules every minute.
func (e *Event) EveryMinute() *Event { return e.Cron("* * * * *") }

// EveryFiveMinutes schedules every 5 minutes.
func (e *Event) EveryFiveMinutes() *Event { return e.Cron("*/5 * * * *") }

// EveryTenMinutes schedules every 10 minutes.
func (e *Event) EveryTenMinutes() *Event { return e.Cron("*/10 * * * *") }

// EveryThirtyMinutes schedules every 30 minutes.
func (e *Event) EveryThirtyMinutes() *Event { return e.Cron("*/30 * * * *") }

// Hourly schedules at minute 0.
func (e *Event) Hourly() *Event { return e.Cron("0 * * * *") }

// Daily schedules at 00:00.
func (e *Event) Daily() *Event { return e.Cron("0 0 * * *") }

// Weekly schedules on Sunday 00:00.
func (e *Event) Weekly() *Event { return e.Cron("0 0 * * 0") }

// Monthly schedules on day 1 at 00:00.
func (e *Event) Monthly() *Event { return e.Cron("0 0 1 * *") }

// DailyAt schedules daily at HH:MM.
func (e *Event) DailyAt(at string) *Event {
	var hour, minute int
	_, err := fmt.Sscanf(at, "%d:%d", &hour, &minute)
	if err != nil {
		return e.Daily()
	}
	return e.Cron(fmt.Sprintf("%d %d * * *", minute, hour))
}

// Weekdays schedules Monday-Friday at 00:00.
func (e *Event) Weekdays() *Event { return e.Cron("0 0 * * 1-5") }

// Sundays schedules Sunday at 00:00.
func (e *Event) Sundays() *Event { return e.Cron("0 0 * * 0") }

// Mondays schedules Monday at 00:00.
func (e *Event) Mondays() *Event { return e.Cron("0 0 * * 1") }

// Events returns registered events.
func (s *Scheduler) Events() []*Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Event, len(s.events))
	copy(out, s.events)
	return out
}

// RunDue runs all events due at the given time.
func (s *Scheduler) RunDue(now time.Time) []error {
	var errs []error
	for _, event := range s.Events() {
		if !event.IsDue(now) {
			continue
		}
		name := event.DisplayName()
		fmt.Printf("Running scheduled event: %s\n", name)

		var unlock func()
		if event.withoutOverlapping {
			ok, unlockFn := event.acquireLock()
			if !ok {
				fmt.Printf("Skipping overlapping event: %s\n", name)
				continue
			}
			unlock = unlockFn
		}

		if err := event.run(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
		if unlock != nil {
			unlock()
		}
	}
	return errs
}

func (e *Event) run() error {
	err := e.callback()
	if e.outputPath != "" {
		_ = os.MkdirAll(filepath.Dir(e.outputPath), 0o700)
		_ = os.Chmod(filepath.Dir(e.outputPath), 0o700)
		line := fmt.Sprintf("%s event=%s err=%v\n", time.Now().Format(time.RFC3339), e.DisplayName(), err)
		f, openErr := os.OpenFile(e.outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if openErr == nil {
			_, _ = f.WriteString(line)
			_ = f.Close()
		}
	}
	return err
}

func (e *Event) acquireLock() (bool, func()) {
	dir := e.mutexPath
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "zatrano-schedule")
	}
	_ = os.MkdirAll(dir, 0o700)
	_ = os.Chmod(dir, 0o700)
	name := e.name
	if name == "" {
		name = "event"
	}
	path := filepath.Join(dir, sanitizeLock(name)+".lock")
	if info, err := os.Stat(path); err == nil {
		if e.overlapExpires > 0 && time.Since(info.ModTime()) > e.overlapExpires {
			_ = os.Remove(path)
		} else {
			return false, func() {}
		}
	}
	if err := os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o600); err != nil {
		return false, func() {}
	}
	return true, func() { _ = os.Remove(path) }
}

func sanitizeLock(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

// IsDue reports whether the event should run at now.
func (e *Event) IsDue(now time.Time) bool {
	if e.timezone != nil {
		now = now.In(e.timezone)
	}
	expr, err := cron.Parse(e.expression)
	if err != nil {
		return false
	}
	return expr.Matches(now)
}

func stringsEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
