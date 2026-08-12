package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Job is a unit of queued work.
type Job interface {
	Handle() error
}

// HandlerFunc adapts a function into a Job.
type HandlerFunc func() error

// Handle runs the function.
func (f HandlerFunc) Handle() error { return f() }

// NamedJob is a serializable job payload.
type NamedJob struct {
	Name    string         `json:"name"`
	Payload map[string]any `json:"payload"`
}

// Queue pushes and pops jobs.
type Queue interface {
	Push(job NamedJob, delay ...time.Duration) error
	Pop() (*ReservedJob, error)
	Size() (int, error)
	Clear() error
}

// ReservedJob is a job reserved for processing.
type ReservedJob struct {
	ID      int64
	Job     NamedJob
	release func(delay time.Duration) error
	delete  func() error
}

// Delete removes the reserved job.
func (j *ReservedJob) Delete() error {
	if j.delete != nil {
		return j.delete()
	}
	return nil
}

// Release releases the job back onto the queue.
func (j *ReservedJob) Release(delay time.Duration) error {
	if j.release != nil {
		return j.release(delay)
	}
	return nil
}

// SyncQueue runs jobs immediately.
type SyncQueue struct {
	handlers map[string]func(map[string]any) error
	mu       sync.RWMutex
}

// NewSyncQueue creates a sync queue.
func NewSyncQueue() *SyncQueue {
	return &SyncQueue{handlers: make(map[string]func(map[string]any) error)}
}

// Register registers a job handler by name.
func (q *SyncQueue) Register(name string, handler func(map[string]any) error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[name] = handler
}

// Push executes the job immediately.
func (q *SyncQueue) Push(job NamedJob, delay ...time.Duration) error {
	if len(delay) > 0 && delay[0] > 0 {
		time.Sleep(delay[0])
	}
	q.mu.RLock()
	handler, ok := q.handlers[job.Name]
	q.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no handler registered for job [%s]", job.Name)
	}
	return handler(job.Payload)
}

// Pop is unsupported for sync queue.
func (q *SyncQueue) Pop() (*ReservedJob, error) { return nil, sql.ErrNoRows }

// Size always returns 0 for sync queue.
func (q *SyncQueue) Size() (int, error) { return 0, nil }

// Clear is a no-op for sync queue.
func (q *SyncQueue) Clear() error { return nil }

// DatabaseQueue stores jobs in a database table.
type DatabaseQueue struct {
	db       *sql.DB
	table    string
	driver   string
	handlers map[string]func(map[string]any) error
	mu       sync.RWMutex
}

// NewDatabaseQueue creates a database-backed queue.
// Optional driver (sqlite|mysql|pgsql) selects dialect-aware SQL.
func NewDatabaseQueue(db *sql.DB, table string, driver ...string) *DatabaseQueue {
	if table == "" {
		table = "jobs"
	}
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &DatabaseQueue{
		db:       db,
		table:    table,
		driver:   d,
		handlers: make(map[string]func(map[string]any) error),
	}
}

// EnsureTable creates the jobs table if needed.
func (q *DatabaseQueue) EnsureTable() error {
	var sqlStr string
	switch q.driver {
	case "pgsql", "postgres", "postgresql":
		sqlStr = fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id BIGSERIAL PRIMARY KEY,
	queue VARCHAR(255) NOT NULL DEFAULT 'default',
	payload TEXT NOT NULL,
	available_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL,
	reserved_at TIMESTAMP NULL
)`, q.table)
	case "mysql":
		sqlStr = fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
	queue VARCHAR(255) NOT NULL DEFAULT 'default',
	payload TEXT NOT NULL,
	available_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL,
	reserved_at DATETIME NULL
)`, q.table)
	default:
		sqlStr = fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	queue VARCHAR(255) NOT NULL DEFAULT 'default',
	payload TEXT NOT NULL,
	available_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL,
	reserved_at DATETIME NULL
)`, q.table)
	}
	_, err := q.db.Exec(sqlStr)
	return err
}

// Register registers a job handler.
func (q *DatabaseQueue) Register(name string, handler func(map[string]any) error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[name] = handler
}

// Handler returns a registered handler.
func (q *DatabaseQueue) Handler(name string) (func(map[string]any) error, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	handler, ok := q.handlers[name]
	return handler, ok
}

// Push stores a job.
func (q *DatabaseQueue) Push(job NamedJob, delay ...time.Duration) error {
	wait := time.Duration(0)
	if len(delay) > 0 {
		wait = delay[0]
	}
	raw, err := json.Marshal(job)
	if err != nil {
		return err
	}
	now := time.Now()
	available := now.Add(wait)
	_, err = q.db.Exec(
		q.q(fmt.Sprintf(`INSERT INTO %s (queue, payload, available_at, created_at) VALUES (?, ?, ?, ?)`, q.table)),
		"default", string(raw), available.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"),
	)
	return err
}

// Pop reserves the next available job.
func (q *DatabaseQueue) Pop() (*ReservedJob, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	row := q.db.QueryRow(q.q(fmt.Sprintf(`
SELECT id, payload FROM %s
WHERE reserved_at IS NULL AND available_at <= ?
ORDER BY id ASC LIMIT 1`, q.table)), now)

	var id int64
	var payload string
	if err := row.Scan(&id, &payload); err != nil {
		return nil, err
	}

	_, err := q.db.Exec(q.q(fmt.Sprintf(`UPDATE %s SET reserved_at = ? WHERE id = ?`, q.table)), now, id)
	if err != nil {
		return nil, err
	}

	var job NamedJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return nil, err
	}

	return &ReservedJob{
		ID:  id,
		Job: job,
		delete: func() error {
			_, err := q.db.Exec(q.q(fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, q.table)), id)
			return err
		},
		release: func(delay time.Duration) error {
			available := time.Now().Add(delay).Format("2006-01-02 15:04:05")
			_, err := q.db.Exec(
				q.q(fmt.Sprintf(`UPDATE %s SET reserved_at = NULL, available_at = ? WHERE id = ?`, q.table)),
				available, id,
			)
			return err
		},
	}, nil
}

// Size returns pending jobs count.
func (q *DatabaseQueue) Size() (int, error) {
	var count int
	err := q.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE reserved_at IS NULL`, q.table)).Scan(&count)
	return count, err
}

// Clear deletes all jobs.
func (q *DatabaseQueue) Clear() error {
	_, err := q.db.Exec(fmt.Sprintf(`DELETE FROM %s`, q.table))
	return err
}

func (q *DatabaseQueue) q(query string) string {
	switch q.driver {
	case "pgsql", "postgres", "postgresql":
		out := make([]byte, 0, len(query)+8)
		n := 1
		for i := 0; i < len(query); i++ {
			if query[i] == '?' {
				out = append(out, '$')
				out = append(out, fmt.Sprintf("%d", n)...)
				n++
				continue
			}
			out = append(out, query[i])
		}
		return string(out)
	default:
		return query
	}
}

// Manager resolves queues and registers handlers.
type Manager struct {
	defaultQueue string
	queues       map[string]Queue
	handlers     map[string]func(map[string]any) error
	mu           sync.RWMutex
	failedMu     sync.Mutex
	failed       []FailedJob
}

// NewManager creates a queue manager.
func NewManager(defaultQueue string, queues map[string]Queue) *Manager {
	return &Manager{
		defaultQueue: defaultQueue,
		queues:       queues,
		handlers:     make(map[string]func(map[string]any) error),
	}
}

// Queue returns a named queue.
func (m *Manager) Queue(name ...string) Queue {
	queueName := m.defaultQueue
	if len(name) > 0 && name[0] != "" {
		queueName = name[0]
	}
	return m.queues[queueName]
}

// Register registers a job handler on all capable queues.
func (m *Manager) Register(name string, handler func(map[string]any) error) {
	m.mu.Lock()
	m.handlers[name] = handler
	m.mu.Unlock()

	for _, q := range m.queues {
		switch typed := q.(type) {
		case *SyncQueue:
			typed.Register(name, handler)
		case *DatabaseQueue:
			typed.Register(name, handler)
		case *RedisQueue:
			typed.Register(name, handler)
		}
	}
}

// Push pushes a named job onto the default queue.
func (m *Manager) Push(name string, payload map[string]any, delay ...time.Duration) error {
	return m.Queue().Push(NamedJob{Name: name, Payload: payload}, delay...)
}

// Handler returns a registered handler.
func (m *Manager) Handler(name string) (func(map[string]any) error, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	handler, ok := m.handlers[name]
	return handler, ok
}

// Work pops and processes one job.
func (m *Manager) Work(queueName ...string) error {
	q := m.Queue(queueName...)
	job, err := q.Pop()
	if err != nil {
		return err
	}
	handler, ok := m.Handler(job.Job.Name)
	if !ok {
		switch typed := q.(type) {
		case *DatabaseQueue:
			handler, ok = typed.Handler(job.Job.Name)
		case *RedisQueue:
			handler, ok = typed.Handler(job.Job.Name)
		}
	}
	if !ok {
		_ = job.Release(5 * time.Second)
		return fmt.Errorf("no handler for job [%s]", job.Job.Name)
	}
	if err := handler(job.Job.Payload); err != nil {
		m.Fail(job.Job, err)
		_ = job.Release(5 * time.Second)
		return err
	}
	return job.Delete()
}
