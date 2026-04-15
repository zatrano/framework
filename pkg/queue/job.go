package queue

import (
	"context"
	"encoding/json"
	"time"
)

// Job is the interface that all background jobs must implement.
type Job interface {
	// Handle executes the job logic. Return an error to trigger a retry.
	Handle(ctx context.Context) error

	// Name returns a unique identifier for this job type (used for serialization).
	Name() string

	// Queue returns the queue name this job should be dispatched to.
	// Return "" for the default queue.
	Queue() string

	// Retries returns how many times this job should be retried on failure.
	// Return 0 for no retries.
	Retries() int

	// Timeout returns the maximum duration Handle may run before being cancelled.
	Timeout() time.Duration
}

// BaseJob provides sensible defaults for the Job interface.
// Embed it in your job structs to only override what you need.
//
//	type SendEmailJob struct {
//	    queue.BaseJob
//	    To      string `json:"to"`
//	    Subject string `json:"subject"`
//	}
type BaseJob struct{}

func (BaseJob) Name() string            { return "" }
func (BaseJob) Queue() string           { return "default" }
func (BaseJob) Retries() int            { return 3 }
func (BaseJob) Timeout() time.Duration  { return 60 * time.Second }
func (BaseJob) Handle(_ context.Context) error { return nil }

// Payload is the serialised envelope stored in the queue backend.
type Payload struct {
	ID        string          `json:"id"`
	JobName   string          `json:"job_name"`
	Data      json.RawMessage `json:"data"`
	Queue     string          `json:"queue"`
	Attempts  int             `json:"attempts"`
	MaxRetry  int             `json:"max_retry"`
	Timeout   time.Duration   `json:"timeout"`
	Delay     time.Duration   `json:"delay,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
