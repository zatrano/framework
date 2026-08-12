package social_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/social"
)

type memPersist struct {
	users    map[int64]memUser
	byEmail  map[string]int64
	accounts map[string]int64 // provider|id -> userID
	nextID   int64
}

type memUser struct {
	Name, Email, Avatar string
	Verified            bool
}

func newMemPersist() *memPersist {
	return &memPersist{
		users:    map[int64]memUser{},
		byEmail:  map[string]int64{},
		accounts: map[string]int64{},
		nextID:   1,
	}
}

func (m *memPersist) key(provider, id string) string {
	return strings.ToLower(provider) + "|" + id
}

func (m *memPersist) FindUserIDByProvider(provider, providerID string) (int64, error) {
	return m.accounts[m.key(provider, providerID)], nil
}

func (m *memPersist) FindUserIDByEmail(email string) (int64, error) {
	return m.byEmail[strings.ToLower(email)], nil
}

func (m *memPersist) CreateUser(name, email, avatar string, emailVerified bool) (int64, error) {
	id := m.nextID
	m.nextID++
	m.users[id] = memUser{Name: name, Email: email, Avatar: avatar, Verified: emailVerified}
	m.byEmail[strings.ToLower(email)] = id
	return id, nil
}

func (m *memPersist) SyncUser(userID int64, name, avatar string, emailVerified bool) error {
	u := m.users[userID]
	if name != "" {
		u.Name = name
	}
	if avatar != "" {
		u.Avatar = avatar
	}
	if emailVerified {
		u.Verified = true
	}
	m.users[userID] = u
	return nil
}

func (m *memPersist) UpsertAccount(userID int64, socialUser *social.User) error {
	m.accounts[m.key(socialUser.Provider, socialUser.ID)] = userID
	return nil
}

func TestPersistCreatesAndLinks(t *testing.T) {
	store := newMemPersist()
	res, err := social.Persist(store, &social.User{
		ID: "g-1", Provider: "google", Email: "a@example.com", Name: "Ada", Avatar: "https://img/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || res.UserID != 1 {
		t.Fatalf("result=%+v", res)
	}
	if store.users[1].Avatar != "https://img/a" || !store.users[1].Verified {
		t.Fatalf("user=%+v", store.users[1])
	}

	res2, err := social.Persist(store, &social.User{
		ID: "g-1", Provider: "google", Email: "a@example.com", Name: "Ada Lovelace", Avatar: "https://img/b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Created || res2.UserID != 1 {
		t.Fatalf("second=%+v", res2)
	}
	// Name is only filled when empty by app SyncUser; Persist always passes the latest name.
	if store.users[1].Name != "Ada Lovelace" || store.users[1].Avatar != "https://img/b" {
		t.Fatalf("synced=%+v", store.users[1])
	}
}

func TestPersistLinksExistingEmail(t *testing.T) {
	store := newMemPersist()
	id, _ := store.CreateUser("Old", "b@example.com", "", false)
	res, err := social.Persist(store, &social.User{
		ID: "gh-9", Provider: "github", Email: "b@example.com", Name: "Bob", Avatar: "pic",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Created || res.UserID != id {
		t.Fatalf("result=%+v want user %d", res, id)
	}
	if store.accounts[store.key("github", "gh-9")] != id {
		t.Fatalf("account not linked")
	}
	if !store.users[id].Verified || store.users[id].Avatar != "pic" {
		t.Fatalf("user=%+v", store.users[id])
	}
}
