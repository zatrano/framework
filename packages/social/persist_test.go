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
		EmailVerified: true,
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
	if store.accounts[store.key("google", "g-1")].Avatar != "https://img/a" {
		t.Fatalf("account snapshot=%+v", store.accounts[store.key("google", "g-1")])
	}

	res2, err := social.Persist(store, &social.User{
		ID: "g-1", Provider: "google", Email: "a@example.com", Name: "Ada Lovelace", Avatar: "https://img/b",
		EmailVerified: true,
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
		EmailVerified: true,
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
		EmailVerified: true,
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
	display := u.Avatar
	if display == "" && acc.Avatar != "" {
		t.Fatal("must not fall back to social account avatar for display")
	}
}

func TestPersistUnverifiedCreatesUnverifiedUser(t *testing.T) {
	store := newMemPersist()
	res, err := social.Persist(store, &social.User{
		ID: "g-new", Provider: "google", Email: "new@example.com",
		EmailVerified: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || store.users[res.UserID].Verified {
		t.Fatalf("expected unverified create: %+v user=%+v", res, store.users[res.UserID])
	}
}

func TestPersistRejectsUnverifiedEmailLinking(t *testing.T) {
	store := newMemPersist()
	victimID, _ := store.CreateUser("Victim", "victim@example.com", "", true)
	_, err := social.Persist(store, &social.User{
		ID: "attacker-id", Provider: "google", Email: "victim@example.com",
		EmailVerified: false,
	})
	if err == nil {
		t.Fatal("expected link rejection")
	}
	if !strings.Contains(err.Error(), "not verified") {
		t.Fatalf("err=%v", err)
	}
	if _, ok := store.accounts[store.key("google", "attacker-id")]; ok {
		t.Fatal("attacker social account must not be linked")
	}
	if store.users[victimID].Verified != true {
		t.Fatal("victim verification must remain unchanged")
	}
}

func TestPersistSeparatesGoogleSubs(t *testing.T) {
	store := newMemPersist()
	resA, err := social.Persist(store, &social.User{
		ID: "google-A", Provider: "google", Email: "user@example.com",
		EmailVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	resB, err := social.Persist(store, &social.User{
		ID: "google-B", Provider: "google", Email: "user@example.com",
		EmailVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resA.UserID != resB.UserID {
		t.Fatalf("same verified email should link to one user: %d vs %d", resA.UserID, resB.UserID)
	}
	if store.accounts[store.key("google", "google-A")].UserID != resA.UserID {
		t.Fatal("missing account A")
	}
	if store.accounts[store.key("google", "google-B")].UserID != resB.UserID {
		t.Fatal("missing account B")
	}
}
