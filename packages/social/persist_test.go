package social_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/social"
)

type memPersist struct {
	users    map[int64]memUser
	byEmail  map[string]int64
	accounts map[string]memAccount // provider|id
	nextID   int64
}

type memUser struct {
	Name, Email, Avatar string
	Verified            bool
}

type memAccount struct {
	UserID int64
	Avatar string
}

func newMemPersist() *memPersist {
	return &memPersist{
		users:    map[int64]memUser{},
		byEmail:  map[string]int64{},
		accounts: map[string]memAccount{},
		nextID:   1,
	}
}

func (m *memPersist) key(provider, id string) string {
	return strings.ToLower(provider) + "|" + id
}

func (m *memPersist) FindUserIDByProvider(provider, providerID string) (int64, error) {
	return m.accounts[m.key(provider, providerID)].UserID, nil
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
	m.accounts[m.key(socialUser.Provider, socialUser.ID)] = memAccount{
		UserID: userID,
		Avatar: strings.TrimSpace(socialUser.Avatar),
	}
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
	// Provider snapshot may exist on the link; canonical display avatar is still the user.
	if store.accounts[store.key("google", "g-1")].Avatar != "https://img/a" {
		t.Fatalf("account snapshot=%+v", store.accounts[store.key("google", "g-1")])
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
	if store.users[1].Name != "Ada Lovelace" || store.users[1].Avatar != "https://img/b" {
		t.Fatalf("synced=%+v", store.users[1])
	}
	if store.accounts[store.key("google", "g-1")].Avatar != "https://img/b" {
		t.Fatalf("account snapshot not updated")
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
	if store.accounts[store.key("github", "gh-9")].UserID != id {
		t.Fatalf("account not linked")
	}
	if !store.users[id].Verified || store.users[id].Avatar != "pic" {
		t.Fatalf("user=%+v", store.users[id])
	}
}

func TestPersistAvatarCanonicalOnUser(t *testing.T) {
	store := newMemPersist()
	_, err := social.Persist(store, &social.User{
		ID: "x", Provider: "google", Email: "c@example.com", Avatar: "https://cdn/provider.png",
	})
	if err != nil {
		t.Fatal(err)
	}
	u := store.users[1]
	acc := store.accounts[store.key("google", "x")]
	if u.Avatar != "https://cdn/provider.png" {
		t.Fatalf("user avatar not set: %+v", u)
	}
	if acc.Avatar != u.Avatar {
		t.Fatalf("snapshot=%q user=%q", acc.Avatar, u.Avatar)
	}
	// Simulate app rule: never prefer account avatar over user for display.
	display := u.Avatar
	if display == "" && acc.Avatar != "" {
		t.Fatal("must not fall back to social account avatar for display")
	}
}
