package orm_test

import (
	"encoding/json"
	"testing"

	"github.com/zatrano/framework/packages/orm"
)

type hiddenUser struct {
	orm.Model
	Email    string `db:"email"`
	Password string `db:"password"`
	Name     string `db:"name"`
}

func (hiddenUser) TableName() string { return "hidden_users" }
func (hiddenUser) Hidden() []string  { return []string{"password"} }

type visibleUser struct {
	orm.Model
	Email string `db:"email"`
	Name  string `db:"name"`
	Bio   string `db:"bio"`
}

func (visibleUser) TableName() string { return "visible_users" }
func (visibleUser) Visible() []string { return []string{"id", "name"} }

func TestToMapHiddenVisible(t *testing.T) {
	u := &hiddenUser{Email: "a@b.c", Password: "secret", Name: "Ada"}
	u.ID = 7
	m := orm.ToMap(u)
	if _, ok := m["password"]; ok {
		t.Fatalf("password should be hidden: %v", m)
	}
	if m["email"] != "a@b.c" || m["name"] != "Ada" {
		t.Fatalf("map=%v", m)
	}

	v := &visibleUser{Email: "x@y.z", Name: "Bob", Bio: "hi"}
	v.ID = 3
	m2 := orm.ToMap(v)
	if len(m2) != 2 || m2["name"] != "Bob" {
		t.Fatalf("visible map=%v", m2)
	}
	if _, ok := m2["email"]; ok {
		t.Fatal("email should not be visible")
	}

	raw, err := orm.ToJSON(u)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["password"]; ok {
		t.Fatalf("json leaked password: %s", raw)
	}
}
