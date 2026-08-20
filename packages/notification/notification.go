package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/view"
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
	ToMail(notifiable Notifiable) *MailMessage
	ToDatabase(notifiable Notifiable) map[string]any
	ToBroadcast(notifiable Notifiable) map[string]any
}

// Channel sends a notification through a transport.
type Channel interface {
	Send(notifiable Notifiable, notification Notification) error
}

// BulkResult summarizes a multi-recipient send.
// For async SendMany, Sent means accepted for delivery (not confirmed delivery).
type BulkResult struct {
	Total  int      `json:"total"`
	Sent   int      `json:"sent"`
	Failed int      `json:"failed"`
	Errors []string `json:"errors,omitempty"`
}

// Manager sends notifications through registered channels.
// Send / SendMany always dispatch asynchronously — callers never wait for transport I/O.
type Manager struct {
	channels map[string]Channel
	store    *Store
	mail     *MailManager
	sms      *SmsManager
	onError  func(error)
	wg       sync.WaitGroup
}

// NewManager creates a notification manager.
func NewManager() *Manager {
	return &Manager{channels: make(map[string]Channel)}
}

// SetMail registers the mail transport and mail notification channel.
// App code should only use Send — not the mail transport directly.
func (m *Manager) SetMail(mailer *MailManager) {
	if m == nil {
		return
	}
	m.mail = mailer
	if mailer != nil {
		m.Extend("mail", NewMailChannel(mailer))
	}
}

// SetMailView attaches the view engine used when rendering mail templates.
func (m *Manager) SetMailView(engine *view.Engine) {
	if m == nil || m.mail == nil {
		return
	}
	m.mail.SetView(engine)
}

// SetSms registers the SMS driver manager, the default "sms" channel, and
// named channels "sms.<driver>" for each registered driver (e.g. sms.twilio).
func (m *Manager) SetSms(sms *SmsManager) {
	if m == nil {
		return
	}
	m.sms = sms
	if sms == nil {
		return
	}
	m.Extend("sms", NewSmsManagerChannel(sms))
	for _, name := range sms.Drivers() {
		m.Extend("sms."+name, NewSmsManagerChannel(sms, name))
	}
}

// Sms returns the SMS driver manager (may be nil).
func (m *Manager) Sms() *SmsManager {
	if m == nil {
		return nil
	}
	return m.sms
}

// Extend registers a channel.
func (m *Manager) Extend(name string, channel Channel) {
	m.channels[name] = channel
}

// SetStore attaches a database notification store for inbox APIs.
func (m *Manager) SetStore(store *Store) {
	m.store = store
}

// SetErrorHandler configures async delivery error reporting (optional).
func (m *Manager) SetErrorHandler(fn func(error)) {
	m.onError = fn
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

// Send queues a notification for async delivery and returns immediately.
func (m *Manager) Send(notifiable Notifiable, notification Notification) error {
	if m == nil {
		return fmt.Errorf("notification manager is nil")
	}
	m.dispatch(func() error {
		return m.SendNow(notifiable, notification)
	})
	return nil
}

// SendNow delivers a notification synchronously (tests / workers).
func (m *Manager) SendNow(notifiable Notifiable, notification Notification) error {
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

// SendMany queues one notification to many recipients (continues on enqueue errors).
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

// Wait blocks until in-flight async deliveries finish (tests).
func (m *Manager) Wait() {
	if m == nil {
		return
	}
	m.wg.Wait()
}

func (m *Manager) dispatch(fn func() error) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			if recovered := recover(); recovered != nil {
				m.report(fmt.Errorf("notification panic: %v", recovered))
			}
		}()
		if err := fn(); err != nil {
			m.report(err)
		}
	}()
}

func (m *Manager) report(err error) {
	if err == nil {
		return
	}
	if m.onError != nil {
		m.onError(err)
		return
	}
	log.Printf("notification: %v", err)
}

// MailChannel sends notifications via mail.
type MailChannel struct {
	mailer *MailManager
}

// NewMailChannel creates a mail notification channel.
func NewMailChannel(mailer *MailManager) *MailChannel {
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
func (Base) ToMail(Notifiable) *MailMessage { return nil }

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
