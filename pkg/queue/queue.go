package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Manager is the high-level queue facade. It dispatches jobs to the
// configured driver and manages job registration for deserialization.
type Manager struct {
	driver   *RedisDriver
	registry map[string]func() Job // job name → factory
}

// New creates a queue manager wrapping the given Redis driver.
func New(driver *RedisDriver) *Manager {
	return &Manager{
		driver:   driver,
		registry: make(map[string]func() Job),
	}
}

// Driver returns the underlying queue driver.
func (m *Manager) Driver() *RedisDriver { return m.driver }

// Register registers a job factory so the worker can deserialize payloads.
// Call this at application startup for every job type.
//
//	queue.Register("send_email", func() queue.Job { return &SendEmailJob{} })
func (m *Manager) Register(name string, factory func() Job) {
	m.registry[name] = factory
}

// Resolve creates a new instance of a registered job by name.
func (m *Manager) Resolve(name string) (Job, bool) {
	factory, ok := m.registry[name]
	if !ok {
		return nil, false
	}
	return factory(), true
}

// ─── Dispatch ──────────────────────────────────────────────────────────────

// Dispatch sends a job to the queue for immediate processing.
//
//	mgr.Dispatch(ctx, &SendEmailJob{To: "user@example.com", Subject: "Hello"})
func (m *Manager) Dispatch(ctx context.Context, job Job) error {
	payload, err := m.marshal(job, 0)
	if err != nil {
		return err
	}
	queueName := job.Queue()
	if queueName == "" {
		queueName = "default"
	}
	return m.driver.Push(ctx, queueName, payload)
}

// Later schedules a job to run after the given delay.
//
//	mgr.Later(ctx, 5*time.Minute, &SendEmailJob{To: "user@example.com"})
func (m *Manager) Later(ctx context.Context, delay time.Duration, job Job) error {
	payload, err := m.marshal(job, delay)
	if err != nil {
		return err
	}
	queueName := job.Queue()
	if queueName == "" {
		queueName = "default"
	}
	return m.driver.LaterAt(ctx, queueName, time.Now().Add(delay), payload)
}

// Size returns the number of pending jobs in the given queue.
func (m *Manager) Size(ctx context.Context, queueName string) (int64, error) {
	return m.driver.Size(ctx, queueName)
}

// Flush removes all pending jobs from the given queue.
func (m *Manager) Flush(ctx context.Context, queueName string) error {
	return m.driver.Flush(ctx, queueName)
}

// ─── Serialization ─────────────────────────────────────────────────────────

func (m *Manager) marshal(job Job, delay time.Duration) ([]byte, error) {
	data, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("queue: marshal job %q: %w", job.Name(), err)
	}
	p := Payload{
		ID:        uuid.New().String(),
		JobName:   job.Name(),
		Data:      data,
		Queue:     job.Queue(),
		Attempts:  0,
		MaxRetry:  job.Retries(),
		Timeout:   job.Timeout(),
		Delay:     delay,
		CreatedAt: time.Now(),
	}
	return json.Marshal(p)
}

// Unmarshal deserializes a payload and resolves the job from the registry.
func (m *Manager) Unmarshal(raw []byte) (*Payload, Job, error) {
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil, fmt.Errorf("queue: unmarshal payload: %w", err)
	}
	job, ok := m.Resolve(p.JobName)
	if !ok {
		return &p, nil, fmt.Errorf("queue: unknown job type %q (not registered)", p.JobName)
	}
	if err := json.Unmarshal(p.Data, job); err != nil {
		return &p, nil, fmt.Errorf("queue: unmarshal job data %q: %w", p.JobName, err)
	}
	return &p, job, nil
}
