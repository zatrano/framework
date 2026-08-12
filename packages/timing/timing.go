package timing

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
)

const attrKey = "server_timing_marks"

type mark struct {
	Name string
	Dur  time.Duration
	Desc string
}

// Add records a named timing mark on the request.
func Add(req *http.Request, name string, dur time.Duration, desc ...string) {
	if req == nil || name == "" {
		return
	}
	m := mark{Name: name, Dur: dur}
	if len(desc) > 0 {
		m.Desc = desc[0]
	}
	existing, _ := req.Get(attrKey).([]mark)
	req.Set(attrKey, append(existing, m))
}

// Measure runs fn and records its duration under name.
func Measure(req *http.Request, name string, fn func()) {
	start := time.Now()
	fn()
	Add(req, name, time.Since(start))
}

// Header builds a Server-Timing header value from marks plus app duration.
func Header(req *http.Request, app time.Duration) string {
	parts := []string{fmt.Sprintf("app;dur=%.2f", float64(app.Microseconds())/1000.0)}
	if req == nil {
		return strings.Join(parts, ", ")
	}
	marks, _ := req.Get(attrKey).([]mark)
	for _, m := range marks {
		part := fmt.Sprintf("%s;dur=%.2f", sanitize(m.Name), float64(m.Dur.Microseconds())/1000.0)
		if m.Desc != "" {
			part = fmt.Sprintf("%s;desc=%q;dur=%.2f", sanitize(m.Name), m.Desc, float64(m.Dur.Microseconds())/1000.0)
		}
		parts = append(parts, part)
	}
	parts = append(parts, fmt.Sprintf("total;dur=%.2f", float64(app.Microseconds())/1000.0))
	return strings.Join(parts, ", ")
}

func sanitize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ",", "_")
	name = strings.ReplaceAll(name, ";", "_")
	return name
}

// Tracker is a simple stopwatch helper.
type Tracker struct {
	mu    sync.Mutex
	start time.Time
	marks []mark
}

// Start begins a tracker.
func Start() *Tracker {
	return &Tracker{start: time.Now()}
}

// Mark records elapsed since start under name.
func (t *Tracker) Mark(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.marks = append(t.marks, mark{Name: name, Dur: time.Since(t.start)})
}

// Apply copies marks onto the request.
func (t *Tracker) Apply(req *http.Request) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, m := range t.marks {
		Add(req, m.Name, m.Dur)
	}
}
