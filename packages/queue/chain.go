package queue

import (
	"fmt"
	"time"
)

// FailedJob records a job that failed during chain execution or explicit Fail().
type FailedJob struct {
	Job      NamedJob  `json:"job"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
}

// Chain runs named jobs sequentially and stops on the first error.
func (m *Manager) Chain(jobs ...NamedJob) error {
	for _, job := range jobs {
		handler, ok := m.Handler(job.Name)
		if !ok {
			err := fmt.Errorf("no handler for job [%s]", job.Name)
			m.Fail(job, err)
			return err
		}
		if err := handler(job.Payload); err != nil {
			m.Fail(job, err)
			return err
		}
	}
	return nil
}

// PushChain enqueues jobs in order onto the default queue.
func (m *Manager) PushChain(jobs ...NamedJob) error {
	for _, job := range jobs {
		if err := m.Queue().Push(job); err != nil {
			m.Fail(job, err)
			return err
		}
	}
	return nil
}

// Fail records a failed job.
func (m *Manager) Fail(job NamedJob, err error) {
	if m == nil {
		return
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	m.failedMu.Lock()
	defer m.failedMu.Unlock()
	m.failed = append(m.failed, FailedJob{
		Job:      job,
		Error:    msg,
		FailedAt: time.Now(),
	})
}

// Failed returns a copy of failed jobs.
func (m *Manager) Failed() []FailedJob {
	m.failedMu.Lock()
	defer m.failedMu.Unlock()
	out := make([]FailedJob, len(m.failed))
	copy(out, m.failed)
	return out
}

// ClearFailed removes recorded failures.
func (m *Manager) ClearFailed() {
	m.failedMu.Lock()
	defer m.failedMu.Unlock()
	m.failed = nil
}
