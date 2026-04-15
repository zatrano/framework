package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WorkerConfig holds worker process configuration.
type WorkerConfig struct {
	// Queues to listen on (default: ["default"]).
	Queues []string

	// MaxTries is the maximum number of attempts before a job is marked as failed.
	// Overridden per-job by Job.Retries() if non-zero.
	MaxTries int

	// Timeout is the default job execution timeout.
	// Overridden per-job by Job.Timeout() if non-zero.
	Timeout time.Duration

	// Sleep is the duration to wait when no jobs are available.
	Sleep time.Duration

	// Logger for worker events.
	Logger *zap.Logger

	// DB for failed job storage (optional — jobs logged and discarded if nil).
	DB *gorm.DB
}

// Worker processes jobs from the queue.
type Worker struct {
	manager *Manager
	config  WorkerConfig
	stop    chan struct{}
	wg      sync.WaitGroup
}

// NewWorker creates a new queue worker.
func NewWorker(manager *Manager, cfg WorkerConfig) *Worker {
	if len(cfg.Queues) == 0 {
		cfg.Queues = []string{"default"}
	}
	if cfg.MaxTries <= 0 {
		cfg.MaxTries = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.Sleep <= 0 {
		cfg.Sleep = 3 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger, _ = zap.NewProduction()
	}

	return &Worker{
		manager: manager,
		config:  cfg,
		stop:    make(chan struct{}),
	}
}

// Run starts the worker loop. Blocks until Stop is called or context is cancelled.
func (w *Worker) Run(ctx context.Context) {
	w.config.Logger.Info("queue worker started",
		zap.Strings("queues", w.config.Queues),
		zap.Int("max_tries", w.config.MaxTries),
		zap.Duration("timeout", w.config.Timeout),
		zap.Duration("sleep", w.config.Sleep),
	)

	w.wg.Add(1)
	go w.migrateLoop(ctx)

	for {
		select {
		case <-ctx.Done():
			w.config.Logger.Info("queue worker stopping (context cancelled)")
			return
		case <-w.stop:
			w.config.Logger.Info("queue worker stopping (stop signal)")
			return
		default:
			w.processNext(ctx)
		}
	}
}

// Stop signals the worker to stop.
func (w *Worker) Stop() {
	close(w.stop)
	w.wg.Wait()
}

// migrateLoop periodically migrates delayed jobs to the ready queue.
func (w *Worker) migrateLoop(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case <-ticker.C:
			for _, q := range w.config.Queues {
				if err := w.manager.driver.MigrateDelayed(ctx, q); err != nil {
					w.config.Logger.Error("delayed migration failed",
						zap.String("queue", q),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// processNext pops and executes the next available job.
func (w *Worker) processNext(ctx context.Context) {
	raw, err := w.manager.driver.Pop(ctx, w.config.Queues...)
	if err != nil {
		w.config.Logger.Error("queue pop error", zap.Error(err))
		time.Sleep(w.config.Sleep)
		return
	}
	if raw == nil {
		// No job available; briefly sleep to avoid tight loop.
		time.Sleep(w.config.Sleep)
		return
	}

	payload, job, err := w.manager.Unmarshal(raw)
	if err != nil {
		w.config.Logger.Error("job unmarshal failed", zap.Error(err))
		if payload != nil {
			w.recordFailure(payload, raw, err, "")
		}
		return
	}

	payload.Attempts++

	maxRetry := payload.MaxRetry
	if jr := job.Retries(); jr > 0 {
		maxRetry = jr
	}
	timeout := payload.Timeout
	if jt := job.Timeout(); jt > 0 {
		timeout = jt
	}

	w.config.Logger.Info("processing job",
		zap.String("id", payload.ID),
		zap.String("job", payload.JobName),
		zap.String("queue", payload.Queue),
		zap.Int("attempt", payload.Attempts),
	)

	execErr := w.executeWithTimeout(ctx, job, timeout)

	if execErr != nil {
		w.config.Logger.Warn("job failed",
			zap.String("id", payload.ID),
			zap.String("job", payload.JobName),
			zap.Int("attempt", payload.Attempts),
			zap.Error(execErr),
		)

		if payload.Attempts < maxRetry {
			// Retry with exponential backoff.
			backoff := time.Duration(math.Pow(2, float64(payload.Attempts))) * time.Second
			w.config.Logger.Info("scheduling retry",
				zap.String("id", payload.ID),
				zap.Duration("backoff", backoff),
				zap.Int("next_attempt", payload.Attempts+1),
			)
			retryPayload, _ := json.Marshal(payload)
			_ = w.manager.driver.LaterAt(ctx, payload.Queue, time.Now().Add(backoff), retryPayload)
		} else {
			// Max retries exceeded — record as failed.
			w.recordFailure(payload, raw, execErr, string(debug.Stack()))
		}
		return
	}

	w.config.Logger.Info("job completed",
		zap.String("id", payload.ID),
		zap.String("job", payload.JobName),
	)
}

// executeWithTimeout runs the job Handle method with a timeout context.
func (w *Worker) executeWithTimeout(parentCtx context.Context, job Job, timeout time.Duration) (retErr error) {
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	// Recover from panics inside Handle.
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	return job.Handle(ctx)
}

// ─── Failed Jobs ───────────────────────────────────────────────────────────

// FailedJob is the GORM model for the failed_jobs table.
type FailedJob struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	JobID      string    `gorm:"size:36;index" json:"job_id"`
	Queue      string    `gorm:"size:255;index" json:"queue"`
	JobName    string    `gorm:"size:255" json:"job_name"`
	Payload    string    `gorm:"type:text" json:"payload"`
	Error      string    `gorm:"type:text" json:"error"`
	StackTrace string    `gorm:"type:text" json:"stack_trace"`
	FailedAt   time.Time `gorm:"autoCreateTime" json:"failed_at"`
}

// TableName overrides the default GORM table name.
func (FailedJob) TableName() string { return "zatrano_failed_jobs" }

// recordFailure stores a failed job in PostgreSQL (or logs it if DB unavailable).
func (w *Worker) recordFailure(payload *Payload, raw []byte, jobErr error, stack string) {
	fj := FailedJob{
		JobID:      payload.ID,
		Queue:      payload.Queue,
		JobName:    payload.JobName,
		Payload:    string(raw),
		Error:      jobErr.Error(),
		StackTrace: stack,
	}

	if w.config.DB != nil {
		if err := w.config.DB.Create(&fj).Error; err != nil {
			w.config.Logger.Error("failed to save failed job",
				zap.String("job_id", payload.ID),
				zap.Error(err),
			)
		} else {
			w.config.Logger.Info("job recorded as failed",
				zap.String("job_id", payload.ID),
				zap.String("job", payload.JobName),
			)
		}
	} else {
		w.config.Logger.Error("job failed (no DB for failed_jobs table)",
			zap.String("job_id", payload.ID),
			zap.String("job", payload.JobName),
			zap.String("error", jobErr.Error()),
		)
	}
}
