package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/queue"
)

// DefaultJobName is the queue job name used by RegisterQueue / PushRun.
const DefaultJobName = "agent.run"

// Pusher is satisfied by *queue.Manager.
type Pusher interface {
	Push(name string, payload map[string]any, delay ...time.Duration) error
}

// RunJob is the serializable payload for an agent queue job.
type RunJob struct {
	Agent   string `json:"agent"`              // Catalog name
	Message string `json:"message"`            // user message
	ID      string `json:"id,omitempty"`       // optional correlation id
	JobName string `json:"job_name,omitempty"` // override queue job name when pushing
}

// RunOutcome is delivered to OnResult after a queued run finishes.
type RunOutcome struct {
	ID      string
	Agent   string
	Message string
	Result  *Result
	Err     error
}

// Catalog maps named agents for queue workers (and optional sync lookup).
type Catalog struct {
	mu     sync.RWMutex
	agents map[string]*Agent
}

// NewCatalog creates an empty agent catalog.
func NewCatalog() *Catalog {
	return &Catalog{agents: make(map[string]*Agent)}
}

// Register stores an agent under name (lowercase trim).
func (c *Catalog) Register(name string, a *Agent) error {
	if c == nil {
		return fmt.Errorf("agent: catalog is nil")
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("agent: name is required")
	}
	if a == nil || a.Chat == nil {
		return fmt.Errorf("agent: agent %q requires Chat", name)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.agents == nil {
		c.agents = make(map[string]*Agent)
	}
	c.agents[name] = a
	return nil
}

// Get returns a registered agent.
func (c *Catalog) Get(name string) (*Agent, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.agents[strings.ToLower(strings.TrimSpace(name))]
	return a, ok
}

// Names returns registered agent names (unsorted).
func (c *Catalog) Names() []string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, 0, len(c.agents))
	for n := range c.agents {
		out = append(out, n)
	}
	return out
}

// Runner binds a Catalog to queue jobs.
type Runner struct {
	Catalog *Catalog
	JobName string // default DefaultJobName
	// Results optionally persists outcomes when RunJob.ID is set.
	Results ResultStore
	// OnResult is called after each queued run (success or failure). Optional.
	OnResult func(RunOutcome)
	// Context factory per job; default context.Background.
	Context func() context.Context
}

// RegisterQueue registers the runner handler on a queue manager.
func (r *Runner) RegisterQueue(m *queue.Manager) error {
	if r == nil || r.Catalog == nil {
		return fmt.Errorf("agent: runner requires Catalog")
	}
	if m == nil {
		return fmt.Errorf("agent: queue manager is nil")
	}
	name := r.jobName()
	m.Register(name, r.Handle)
	return nil
}

func (r *Runner) jobName() string {
	if r != nil && strings.TrimSpace(r.JobName) != "" {
		return strings.TrimSpace(r.JobName)
	}
	return DefaultJobName
}

// Handle is the queue handler for agent.run payloads.
func (r *Runner) Handle(payload map[string]any) error {
	if r == nil || r.Catalog == nil {
		return fmt.Errorf("agent: runner requires Catalog")
	}
	job, err := payloadToRunJob(payload)
	if err != nil {
		return err
	}
	a, ok := r.Catalog.Get(job.Agent)
	if !ok {
		return fmt.Errorf("agent: unknown agent %q", job.Agent)
	}
	ctx := context.Background()
	if r.Context != nil {
		ctx = r.Context()
		if ctx == nil {
			ctx = context.Background()
		}
	}
	res, runErr := a.Run(ctx, job.Message)
	outcome := RunOutcome{
		ID:      job.ID,
		Agent:   job.Agent,
		Message: job.Message,
		Result:  res,
		Err:     runErr,
	}
	if r.Results != nil && job.ID != "" {
		_ = r.Results.Put(ctx, OutcomeToStored(outcome))
	}
	if r.OnResult != nil {
		r.OnResult(outcome)
	}
	return runErr
}

// PushRun enqueues an agent run on pusher (typically *queue.Manager).
func (r *Runner) PushRun(p Pusher, job RunJob, delay ...time.Duration) error {
	if r == nil {
		return fmt.Errorf("agent: runner is nil")
	}
	if p == nil {
		return fmt.Errorf("agent: pusher is nil")
	}
	job.Agent = strings.ToLower(strings.TrimSpace(job.Agent))
	job.Message = strings.TrimSpace(job.Message)
	if job.Agent == "" || job.Message == "" {
		return fmt.Errorf("agent: RunJob agent and message are required")
	}
	if _, ok := r.Catalog.Get(job.Agent); !ok {
		return fmt.Errorf("agent: unknown agent %q", job.Agent)
	}
	name := r.jobName()
	if strings.TrimSpace(job.JobName) != "" {
		name = strings.TrimSpace(job.JobName)
	}
	payload := map[string]any{
		"agent":   job.Agent,
		"message": job.Message,
	}
	if job.ID != "" {
		payload["id"] = job.ID
	}
	return p.Push(name, payload, delay...)
}

// PushRun is a convenience that builds a one-off Runner for a catalog.
func PushRun(p Pusher, catalog *Catalog, job RunJob, delay ...time.Duration) error {
	return (&Runner{Catalog: catalog}).PushRun(p, job, delay...)
}

func payloadToRunJob(payload map[string]any) (RunJob, error) {
	if payload == nil {
		return RunJob{}, fmt.Errorf("agent: empty payload")
	}
	job := RunJob{
		Agent:   stringVal(payload["agent"]),
		Message: stringVal(payload["message"]),
		ID:      stringVal(payload["id"]),
	}
	if job.Agent == "" || job.Message == "" {
		return RunJob{}, fmt.Errorf("agent: payload requires agent and message")
	}
	return job, nil
}

func stringVal(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	default:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
