package factory_test

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/factory"
	"github.com/zatrano/framework/packages/orm"
)

type stateUser struct {
	orm.Model
	Name            string     `db:"name"`
	Email           string     `db:"email"`
	EmailVerifiedAt *time.Time `db:"email_verified_at"`
}

func (stateUser) TableName() string { return "state_users" }

func TestFactoryStates(t *testing.T) {
	factory.ClearStates()
	factory.For[stateUser](func() map[string]any {
		return map[string]any{
			"name":  "Plain",
			"email": "plain@example.test",
		}
	})
	factory.RegisterState[stateUser]("verified", func() map[string]any {
		now := time.Now().UTC()
		return map[string]any{"email_verified_at": now, "name": "Verified"}
	})
	factory.RegisterState[stateUser]("admin", func() map[string]any {
		return map[string]any{"name": "Admin"}
	})

	if !factory.HasState[stateUser]("verified") {
		t.Fatal("expected verified state")
	}

	attrs, err := factory.Of[stateUser]().State("verified", "admin").Merge(map[string]any{
		"email": "combo@example.test",
	}).Make()
	if err != nil {
		t.Fatal(err)
	}
	if attrs["name"] != "Admin" || attrs["email"] != "combo@example.test" || attrs["email_verified_at"] == nil {
		t.Fatalf("attrs=%v", attrs)
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE state_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		email TEXT,
		email_verified_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")

	user, err := factory.Of[stateUser]().State("verified").Create()
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Verified" || user.EmailVerifiedAt == nil {
		t.Fatalf("user=%+v", user)
	}
}
