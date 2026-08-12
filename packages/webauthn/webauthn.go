package webauthn

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	webauthnlib "github.com/go-webauthn/webauthn/webauthn"
)

// CreationOptions is returned by BeginRegistration for the browser ceremony.
type CreationOptions struct {
	ChallengeID string                       `json:"challenge_id"`
	Options     *protocol.CredentialCreation `json:"options"`
}

// RequestOptions is returned by BeginLogin for the browser ceremony.
type RequestOptions struct {
	ChallengeID string                        `json:"challenge_id"`
	Options     *protocol.CredentialAssertion `json:"options"`
}

// Credential is a registered authenticator stored in memory.
type Credential struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
	raw       webauthnlib.Credential
}

// CredentialStore persists WebAuthn credentials (default: in-memory).
type CredentialStore interface {
	CredentialsFor(userID string) []Credential
	Add(userID string, cred Credential) error
	Replace(userID string, creds []Credential) error
}

type memoryStore struct {
	mu    sync.RWMutex
	creds map[string][]Credential
}

func newMemoryStore() *memoryStore {
	return &memoryStore{creds: make(map[string][]Credential)}
}

func (s *memoryStore) CredentialsFor(userID string) []Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Credential, len(s.creds[userID]))
	copy(out, s.creds[userID])
	return out
}

func (s *memoryStore) Add(userID string, cred Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creds[userID] = append(s.creds[userID], cred)
	return nil
}

func (s *memoryStore) Replace(userID string, creds []Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]Credential, len(creds))
	copy(cp, creds)
	s.creds[userID] = cp
	return nil
}

type pendingSession struct {
	UserID   string
	UserName string
	Display  string
	Type     string // registration | authentication
	Session  webauthnlib.SessionData
}

// Manager wraps go-webauthn with an in-memory credential store.
type Manager struct {
	mu       sync.Mutex
	wa       *webauthnlib.WebAuthn
	initErr  error
	rpID     string
	rpOrigin string
	rpName   string
	store    CredentialStore
	sessions map[string]pendingSession
	users    map[string]*userRecord // name/display cache
}

type userRecord struct {
	ID      string
	Name    string
	Display string
}

// New creates a WebAuthn manager. RPID and origin are required for ceremonies;
// missing config yields clear errors from Begin*/Finish* (no accept-any stub).
func New(rpID, rpOrigin, rpDisplayName string) *Manager {
	rpID = strings.TrimSpace(rpID)
	rpOrigin = strings.TrimSpace(rpOrigin)
	rpDisplayName = strings.TrimSpace(rpDisplayName)
	if rpDisplayName == "" {
		rpDisplayName = "ZATRANO"
	}
	m := &Manager{
		rpID:     rpID,
		rpOrigin: rpOrigin,
		rpName:   rpDisplayName,
		store:    newMemoryStore(),
		sessions: make(map[string]pendingSession),
		users:    make(map[string]*userRecord),
	}
	if rpID == "" || rpOrigin == "" {
		m.initErr = fmt.Errorf("webauthn: WEBAUTHN_RP_ID and WEBAUTHN_RP_ORIGIN are required")
		return m
	}
	wa, err := webauthnlib.New(&webauthnlib.Config{
		RPID:          rpID,
		RPDisplayName: rpDisplayName,
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		m.initErr = fmt.Errorf("webauthn: %w", err)
		return m
	}
	m.wa = wa
	return m
}

// SetStore replaces the credential store (nil resets to in-memory).
func (m *Manager) SetStore(store CredentialStore) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if store == nil {
		m.store = newMemoryStore()
		return
	}
	m.store = store
}

// BeginRegistration starts a registration ceremony.
func (m *Manager) BeginRegistration(userID, userName, displayName string) (*CreationOptions, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("webauthn: user id required")
	}
	if userName == "" {
		userName = userID
	}
	if displayName == "" {
		displayName = userName
	}
	user := m.user(userID, userName, displayName)
	creation, session, err := m.wa.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("webauthn: begin registration: %w", err)
	}
	challengeID := randomID()
	m.mu.Lock()
	m.users[userID] = &userRecord{ID: userID, Name: userName, Display: displayName}
	m.sessions[challengeID] = pendingSession{
		UserID:   userID,
		UserName: userName,
		Display:  displayName,
		Type:     "registration",
		Session:  *session,
	}
	m.mu.Unlock()
	return &CreationOptions{ChallengeID: challengeID, Options: creation}, nil
}

// FinishRegistration verifies the authenticator attestation response JSON.
func (m *Manager) FinishRegistration(challengeID string, responseJSON []byte) (*Credential, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	pending, err := m.takeSession(challengeID, "registration")
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(responseJSON)
	if err != nil {
		return nil, fmt.Errorf("webauthn: parse registration response: %w", err)
	}
	user := m.user(pending.UserID, pending.UserName, pending.Display)
	cred, err := m.wa.CreateCredential(user, pending.Session, parsed)
	if err != nil {
		return nil, fmt.Errorf("webauthn: finish registration: %w", err)
	}
	stored := Credential{
		ID:        base64.RawURLEncoding.EncodeToString(cred.ID),
		UserID:    pending.UserID,
		PublicKey: base64.RawURLEncoding.EncodeToString(cred.PublicKey),
		CreatedAt: time.Now().UTC(),
		raw:       *cred,
	}
	if err := m.store.Add(pending.UserID, stored); err != nil {
		return nil, err
	}
	return &stored, nil
}

// BeginLogin starts an authentication ceremony.
func (m *Manager) BeginLogin(userID string) (*RequestOptions, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("webauthn: user id required")
	}
	creds := m.store.CredentialsFor(userID)
	if len(creds) == 0 {
		return nil, fmt.Errorf("webauthn: no credentials for user")
	}
	m.mu.Lock()
	rec := m.users[userID]
	m.mu.Unlock()
	name, display := userID, userID
	if rec != nil {
		name, display = rec.Name, rec.Display
	}
	user := m.user(userID, name, display)
	assertion, session, err := m.wa.BeginLogin(user)
	if err != nil {
		return nil, fmt.Errorf("webauthn: begin login: %w", err)
	}
	challengeID := randomID()
	m.mu.Lock()
	m.sessions[challengeID] = pendingSession{
		UserID:   userID,
		UserName: name,
		Display:  display,
		Type:     "authentication",
		Session:  *session,
	}
	m.mu.Unlock()
	return &RequestOptions{ChallengeID: challengeID, Options: assertion}, nil
}

// FinishLogin verifies the authenticator assertion response JSON.
func (m *Manager) FinishLogin(challengeID string, responseJSON []byte) (bool, error) {
	if err := m.ensure(); err != nil {
		return false, err
	}
	pending, err := m.takeSession(challengeID, "authentication")
	if err != nil {
		return false, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(responseJSON)
	if err != nil {
		return false, fmt.Errorf("webauthn: parse login response: %w", err)
	}
	user := m.user(pending.UserID, pending.UserName, pending.Display)
	cred, err := m.wa.ValidateLogin(user, pending.Session, parsed)
	if err != nil {
		return false, fmt.Errorf("webauthn: finish login: %w", err)
	}
	// Persist updated authenticator counter / flags.
	id := base64.RawURLEncoding.EncodeToString(cred.ID)
	existing := m.store.CredentialsFor(pending.UserID)
	for i := range existing {
		if existing[i].ID == id {
			existing[i].raw = *cred
			existing[i].PublicKey = base64.RawURLEncoding.EncodeToString(cred.PublicKey)
			_ = m.store.Replace(pending.UserID, existing)
			break
		}
	}
	return true, nil
}

// CredentialsFor returns registered credentials for a user.
func (m *Manager) CredentialsFor(userID string) []Credential {
	return m.store.CredentialsFor(userID)
}

func (m *Manager) ensure() error {
	if m.wa != nil {
		return nil
	}
	if m.initErr != nil {
		return m.initErr
	}
	return fmt.Errorf("webauthn: WEBAUTHN_RP_ID and WEBAUTHN_RP_ORIGIN are required")
}

func (m *Manager) takeSession(id, typ string) (pendingSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return pendingSession{}, fmt.Errorf("webauthn: challenge not found")
	}
	delete(m.sessions, id)
	if s.Type != typ {
		return pendingSession{}, fmt.Errorf("webauthn: challenge type mismatch")
	}
	if !s.Session.Expires.IsZero() && time.Now().After(s.Session.Expires) {
		return pendingSession{}, fmt.Errorf("webauthn: challenge expired")
	}
	return s, nil
}

func (m *Manager) user(userID, name, display string) *userEntity {
	creds := m.store.CredentialsFor(userID)
	libCreds := make([]webauthnlib.Credential, 0, len(creds))
	for _, c := range creds {
		libCreds = append(libCreds, c.raw)
	}
	return &userEntity{
		id:      []byte(userID),
		name:    name,
		display: display,
		creds:   libCreds,
	}
}

type userEntity struct {
	id      []byte
	name    string
	display string
	creds   []webauthnlib.Credential
}

func (u *userEntity) WebAuthnID() []byte                            { return u.id }
func (u *userEntity) WebAuthnName() string                          { return u.name }
func (u *userEntity) WebAuthnDisplayName() string                   { return u.display }
func (u *userEntity) WebAuthnCredentials() []webauthnlib.Credential { return u.creds }

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// MarshalJSON omits internal raw credential bytes from API responses.
func (c Credential) MarshalJSON() ([]byte, error) {
	type alias struct {
		ID        string    `json:"id"`
		UserID    string    `json:"user_id"`
		PublicKey string    `json:"public_key"`
		CreatedAt time.Time `json:"created_at"`
	}
	return json.Marshal(alias{
		ID:        c.ID,
		UserID:    c.UserID,
		PublicKey: c.PublicKey,
		CreatedAt: c.CreatedAt,
	})
}
