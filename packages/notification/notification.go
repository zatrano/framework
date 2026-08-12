package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zatrano/framework/packages/mail"
)

// Notifiable can receive notifications.
type Notifiable interface {
	RouteNotificationFor(channel string) string
	NotificationID() any
}

// TypedNotifiable optionally supplies a polymorphic type label.
type TypedNotifiable interface {
	NotifiableType() string
}

// Notification is a message that can be sent through channels.
type Notification interface {
	Via() []string
	ToMail(notifiable Notifiable) *mail.Message
	ToDatabase(notifiable Notifiable) map[string]any
	ToBroadcast(notifiable Notifiable) map[string]any
}

// Channel sends a notification through a transport.
type Channel interface {
	Send(notifiable Notifiable, notification Notification) error
}

// BulkResult summarizes a multi-recipient send.
type BulkResult struct {
	Total  int      `json:"total"`
	Sent   int      `json:"sent"`
	Failed int      `json:"failed"`
	Errors []string `json:"errors,omitempty"`
}

// Manager sends notifications through registered channels.
type Manager struct {
	channels map[string]Channel
	store    *Store
}

// NewManager creates a notification manager.
func NewManager() *Manager {
	return &Manager{channels: make(map[string]Channel)}
}

// Extend registers a channel.
func (m *Manager) Extend(name string, channel Channel) {
	m.channels[name] = channel
}

// SetStore attaches a database notification store for inbox APIs.
func (m *Manager) SetStore(store *Store) {
	m.store = store
}

// Store returns the attached notification store (may be nil).
func (m *Manager) Store() *Store {
	return m.store
}

// Channel returns a registered channel.
func (m *Manager) Channel(name string) (Channel, bool) {
	ch, ok := m.channels[name]
	return ch, ok
}

// Send sends a notification to a notifiable.
func (m *Manager) Send(notifiable Notifiable, notification Notification) error {
	for _, name := range notification.Via() {
		channel, ok := m.channels[name]
		if !ok {
			return fmt.Errorf("notification channel [%s] is not defined", name)
		}
		if err := channel.Send(notifiable, notification); err != nil {
			return err
		}
	}
	return nil
}

// SendMany sends one notification to many recipients (continues on per-recipient errors).
func (m *Manager) SendMany(recipients []Recipient, notification Notification) BulkResult {
	result := BulkResult{Total: len(recipients)}
	for i, recipient := range recipients {
		if err := m.Send(recipient, notification); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d (%v): %v", i+1, recipient.NotificationID(), err))
			continue
		}
		result.Sent++
	}
	return result
}

// MailChannel sends notifications via mail.
type MailChannel struct {
	mailer *mail.Manager
}

// NewMailChannel creates a mail notification channel.
func NewMailChannel(mailer *mail.Manager) *MailChannel {
	return &MailChannel{mailer: mailer}
}

// Send delivers the mail representation.
func (c *MailChannel) Send(notifiable Notifiable, notification Notification) error {
	message := notification.ToMail(notifiable)
	if message == nil {
		return nil
	}
	if len(message.To) == 0 {
		route := notifiable.RouteNotificationFor("mail")
		if route != "" {
			message.To = []string{route}
		}
	}
	return c.mailer.Send(message)
}

// DatabaseChannel stores notifications in the database.
type DatabaseChannel struct {
	db     *sql.DB
	table  string
	driver string
}

// NewDatabaseChannel creates a database notification channel.
// Optional driver (sqlite|mysql|pgsql) rewrites placeholders for PostgreSQL.
func NewDatabaseChannel(db *sql.DB, table string, driver ...string) *DatabaseChannel {
	if table == "" {
		table = "notifications"
	}
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &DatabaseChannel{db: db, table: table, driver: d}
}

// Send stores the notification payload.
func (c *DatabaseChannel) Send(notifiable Notifiable, notification Notification) error {
	payload := notification.ToDatabase(notifiable)
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	typ := "recipient"
	if t, ok := notifiable.(TypedNotifiable); ok && t.NotifiableType() != "" {
		typ = t.NotifiableType()
	}
	_, err = c.db.Exec(
		c.q(fmt.Sprintf(`INSERT INTO %s (notifiable_type, notifiable_id, type, data, created_at) VALUES (?, ?, ?, ?, ?)`, c.table)),
		typ,
		fmt.Sprint(notifiable.NotificationID()),
		fmt.Sprintf("%T", notification),
		string(raw),
		time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	return err
}

func (c *DatabaseChannel) q(query string) string {
	return rewritePlaceholders(c.driver, query)
}

// BroadcastChannel publishes notifications to a broadcaster.
type BroadcastChannel struct {
	broadcaster Broadcaster
}

// Broadcaster publishes events to channels.
type Broadcaster interface {
	Broadcast(channel string, event string, payload map[string]any) error
}

// NewBroadcastChannel creates a broadcast notification channel.
func NewBroadcastChannel(broadcaster Broadcaster) *BroadcastChannel {
	return &BroadcastChannel{broadcaster: broadcaster}
}

// Send broadcasts the notification.
func (c *BroadcastChannel) Send(notifiable Notifiable, notification Notification) error {
	payload := notification.ToBroadcast(notifiable)
	if payload == nil {
		payload = map[string]any{}
	}
	channel := notifiable.RouteNotificationFor("broadcast")
	if channel == "" {
		channel = fmt.Sprintf("users.%v", notifiable.NotificationID())
	}
	return c.broadcaster.Broadcast(channel, fmt.Sprintf("%T", notification), payload)
}

// Base can be embedded to provide empty channel representations.
type Base struct{}

// ToMail returns nil by default.
func (Base) ToMail(Notifiable) *mail.Message { return nil }

// ToDatabase returns nil by default.
func (Base) ToDatabase(Notifiable) map[string]any { return nil }

// ToBroadcast returns nil by default.
func (Base) ToBroadcast(Notifiable) map[string]any { return nil }

func rewritePlaceholders(driver, query string) string {
	switch driver {
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
