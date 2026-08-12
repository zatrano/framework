package notification_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/mail"
	"github.com/zatrano/framework/packages/notification"
	_ "modernc.org/sqlite"
)

func TestSendMailDatabaseSms(t *testing.T) {
	db := openNotifDB(t)
	defer db.Close()

	mailer := mail.NewManager("log", "hello@example.com", "ZATRANO", map[string]mail.Mailer{
		"log": stubMailer{},
	})
	sms := &notification.MemorySmsSender{}
	mgr := notification.NewManager()
	mgr.Extend("mail", notification.NewMailChannel(mailer))
	mgr.Extend("database", notification.NewDatabaseChannel(db, "notifications", "sqlite"))
	mgr.Extend("sms", notification.NewSmsChannel(sms, "ZATRANO"))
	mgr.SetStore(notification.NewStore(db, "notifications", "sqlite"))

	rec := notification.Recipient{ID: "u1", Email: "a@example.com", Phone: "+905551112233"}
	msg := notification.Message{
		Channels: []string{"mail", "database", "sms"},
		Subject:  "Hello",
		Body:     "World",
	}
	if err := mgr.Send(rec, msg); err != nil {
		t.Fatal(err)
	}
	if _, ok := sms.Last(); !ok {
		t.Fatal("expected sms")
	}
	items, err := mgr.Store().ListFor("u1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(items))
	}
}

type stubMailer struct{}

func (stubMailer) Send(*mail.Message) error { return nil }

func TestSendManyAndCSVImport(t *testing.T) {
	db := openNotifDB(t)
	defer db.Close()
	mgr := notification.NewManager()
	mgr.Extend("database", notification.NewDatabaseChannel(db, "notifications", "sqlite"))

	csvData := "id,email,phone\n1,a@x.com,+111\n2,b@x.com,+222\n"
	recipients, err := notification.ImportRecipientsBytes("people.csv", []byte(csvData))
	if err != nil {
		t.Fatal(err)
	}
	if len(recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(recipients))
	}
	result := mgr.SendMany(recipients, notification.Message{
		Channels: []string{"database"},
		Subject:  "Bulk",
		Body:     "Hi",
	})
	if result.Sent != 2 || result.Failed != 0 {
		t.Fatalf("unexpected bulk result: %+v", result)
	}
}

func TestRecipientFromMapRequiresContact(t *testing.T) {
	_, err := notification.RecipientFromMap(map[string]string{"name": "Ada"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "id, email, or phone") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func openNotifDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:notif?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
CREATE TABLE notifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  notifiable_type TEXT NOT NULL DEFAULT 'recipient',
  notifiable_id TEXT NOT NULL,
  type TEXT NOT NULL,
  data TEXT NOT NULL,
  read_at TEXT NULL,
  created_at TEXT NULL,
  updated_at TEXT NULL
)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
