package queue

import (
	"encoding/json"
	"fmt"
	"time"
)

// JobRow is a dashboard-friendly job record.
type JobRow struct {
	ID          int64          `json:"id"`
	Queue       string         `json:"queue"`
	Name        string         `json:"name"`
	Payload     map[string]any `json:"payload"`
	AvailableAt string         `json:"available_at"`
	CreatedAt   string         `json:"created_at"`
	ReservedAt  *string        `json:"reserved_at"`
	Status      string         `json:"status"`
}

// Stats summarizes queue health for the dashboard.
type Stats struct {
	Connection string         `json:"connection"`
	Pending    int            `json:"pending"`
	Reserved   int            `json:"reserved"`
	Total      int            `json:"total"`
	Handlers   []string       `json:"handlers"`
	Queues     map[string]int `json:"queues"`
}

// Stats returns manager-level queue stats.
func (m *Manager) Stats() Stats {
	stats := Stats{
		Connection: m.defaultQueue,
		Handlers:   make([]string, 0),
		Queues:     map[string]int{},
	}
	m.mu.RLock()
	for name := range m.handlers {
		stats.Handlers = append(stats.Handlers, name)
	}
	m.mu.RUnlock()

	for name, q := range m.queues {
		size, _ := q.Size()
		stats.Queues[name] = size
		if name == m.defaultQueue {
			stats.Pending = size
		}
	}

	if dbq, ok := m.Queue("database").(*DatabaseQueue); ok && dbq != nil {
		pending, reserved, total := dbq.counts()
		stats.Pending = pending
		stats.Reserved = reserved
		stats.Total = total
	} else if dbq, ok := m.Queue().(*DatabaseQueue); ok && dbq != nil {
		pending, reserved, total := dbq.counts()
		stats.Pending = pending
		stats.Reserved = reserved
		stats.Total = total
	}
	return stats
}

// ListJobs returns recent database jobs when available.
func (m *Manager) ListJobs(limit int) ([]JobRow, error) {
	if limit <= 0 {
		limit = 50
	}
	dbq, ok := m.Queue("database").(*DatabaseQueue)
	if !ok || dbq == nil {
		dbq, ok = m.Queue().(*DatabaseQueue)
		if !ok || dbq == nil {
			return []JobRow{}, nil
		}
	}
	return dbq.List(limit)
}

// DeleteJob removes a database job by id.
func (m *Manager) DeleteJob(id int64) error {
	dbq, ok := m.Queue("database").(*DatabaseQueue)
	if !ok || dbq == nil {
		dbq, ok = m.Queue().(*DatabaseQueue)
		if !ok || dbq == nil {
			return fmt.Errorf("database queue is not configured")
		}
	}
	_, err := dbq.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, dbq.table), id)
	return err
}

// RetryJob makes a reserved/failed job available immediately.
func (m *Manager) RetryJob(id int64) error {
	dbq, ok := m.Queue("database").(*DatabaseQueue)
	if !ok || dbq == nil {
		dbq, ok = m.Queue().(*DatabaseQueue)
		if !ok || dbq == nil {
			return fmt.Errorf("database queue is not configured")
		}
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := dbq.db.Exec(
		fmt.Sprintf(`UPDATE %s SET reserved_at = NULL, available_at = ? WHERE id = ?`, dbq.table),
		now, id,
	)
	return err
}

func (q *DatabaseQueue) counts() (pending, reserved, total int) {
	_ = q.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE reserved_at IS NULL`, q.table)).Scan(&pending)
	_ = q.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE reserved_at IS NOT NULL`, q.table)).Scan(&reserved)
	_ = q.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, q.table)).Scan(&total)
	return
}

// List returns recent jobs.
func (q *DatabaseQueue) List(limit int) ([]JobRow, error) {
	rows, err := q.db.Query(fmt.Sprintf(`
SELECT id, queue, payload, available_at, created_at, reserved_at
FROM %s ORDER BY id DESC LIMIT ?`, q.table), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]JobRow, 0)
	for rows.Next() {
		var row JobRow
		var payload string
		var reserved *string
		if err := rows.Scan(&row.ID, &row.Queue, &payload, &row.AvailableAt, &row.CreatedAt, &reserved); err != nil {
			return nil, err
		}
		row.ReservedAt = reserved
		var job NamedJob
		_ = json.Unmarshal([]byte(payload), &job)
		row.Name = job.Name
		row.Payload = job.Payload
		if reserved != nil && *reserved != "" {
			row.Status = "reserved"
		} else {
			row.Status = "pending"
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
