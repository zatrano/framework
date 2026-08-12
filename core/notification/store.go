package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Record is a stored in-app notification.
type Record struct {
	ID             int64          `json:"id"`
	NotifiableType string         `json:"notifiable_type"`
	NotifiableID   string         `json:"notifiable_id"`
	Type           string         `json:"type"`
	Data           map[string]any `json:"data"`
	ReadAt         *time.Time     `json:"read_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// Store reads and updates database notifications (inbox).
type Store struct {
	db     *sql.DB
	table  string
	driver string
}

// NewStore creates a notification store.
func NewStore(db *sql.DB, table string, driver ...string) *Store {
	if table == "" {
		table = "notifications"
	}
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &Store{db: db, table: table, driver: d}
}

// ListFor returns notifications for a notifiable (newest first).
func (s *Store) ListFor(notifiableID string, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(s.q(fmt.Sprintf(`
SELECT id, notifiable_type, notifiable_id, type, data, read_at, created_at
FROM %s WHERE notifiable_id = ?
ORDER BY id DESC LIMIT ?`, s.table)), notifiableID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// UnreadFor returns unread notifications.
func (s *Store) UnreadFor(notifiableID string, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(s.q(fmt.Sprintf(`
SELECT id, notifiable_type, notifiable_id, type, data, read_at, created_at
FROM %s WHERE notifiable_id = ? AND read_at IS NULL
ORDER BY id DESC LIMIT ?`, s.table)), notifiableID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// MarkAsRead marks a single notification as read.
func (s *Store) MarkAsRead(id int64, notifiableID string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(s.q(fmt.Sprintf(`
UPDATE %s SET read_at = ? WHERE id = ? AND notifiable_id = ? AND read_at IS NULL`, s.table)),
		now, id, notifiableID)
	return err
}

// MarkAllRead marks all notifications for a notifiable as read.
func (s *Store) MarkAllRead(notifiableID string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(s.q(fmt.Sprintf(`
UPDATE %s SET read_at = ? WHERE notifiable_id = ? AND read_at IS NULL`, s.table)),
		now, notifiableID)
	return err
}

func (s *Store) q(query string) string {
	return rewritePlaceholders(s.driver, query)
}

func scanRecords(rows *sql.Rows) ([]Record, error) {
	out := make([]Record, 0)
	for rows.Next() {
		var (
			rec      Record
			dataRaw  string
			readRaw  sql.NullString
			created  string
			typeName sql.NullString
		)
		if err := rows.Scan(&rec.ID, &typeName, &rec.NotifiableID, &rec.Type, &dataRaw, &readRaw, &created); err != nil {
			return nil, err
		}
		if typeName.Valid {
			rec.NotifiableType = typeName.String
		}
		_ = json.Unmarshal([]byte(dataRaw), &rec.Data)
		if rec.Data == nil {
			rec.Data = map[string]any{}
		}
		if readRaw.Valid && readRaw.String != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", readRaw.String); err == nil {
				rec.ReadAt = &t
			}
		}
		if t, err := time.Parse("2006-01-02 15:04:05", created); err == nil {
			rec.CreatedAt = t
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
