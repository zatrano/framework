package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/packages/support/uuid"
)

// Client is an OAuth2 client application.
type Client struct {
	ID           string   `json:"id"`
	Secret       string   `json:"secret,omitempty"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

// AccessToken is an issued OAuth2 token response.
type AccessToken struct {
	Token        string    `json:"access_token"`
	Type         string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ClientID     string    `json:"client_id"`
	UserID       string    `json:"user_id,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Server is a minimal OAuth2 authorization server.
// Clients and tokens persist to OAUTH_STORE_PATH when set; otherwise memory only.
type Server struct {
	mu         sync.Mutex
	clients    map[string]*Client
	codes      map[string]authCode
	tokens     map[string]*AccessToken
	refresh    map[string]*refreshRecord
	ttl        time.Duration
	refreshTTL time.Duration
	storePath  string
}

type authCode struct {
	ClientID            string    `json:"client_id"`
	RedirectURI         string    `json:"redirect_uri"`
	UserID              string    `json:"user_id"`
	Scope               string    `json:"scope"`
	CodeChallenge       string    `json:"code_challenge"`
	CodeChallengeMethod string    `json:"code_challenge_method"`
	ExpiresAt           time.Time `json:"expires_at"`
}

type refreshRecord struct {
	Token     string    `json:"token"`
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	Scope     string    `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
}

type persistState struct {
	Clients map[string]*Client        `json:"clients"`
	Tokens  map[string]*AccessToken   `json:"tokens"`
	Refresh map[string]*refreshRecord `json:"refresh"`
}

// New creates an in-memory OAuth2 server.
func New() *Server {
	return &Server{
		clients:    make(map[string]*Client),
		codes:      make(map[string]authCode),
		tokens:     make(map[string]*AccessToken),
		refresh:    make(map[string]*refreshRecord),
		ttl:        time.Hour,
		refreshTTL: 30 * 24 * time.Hour,
	}
}

// NewWithStore creates a server that loads/saves clients and tokens from path.
func NewWithStore(path string) (*Server, error) {
	s := New()
	s.storePath = strings.TrimSpace(path)
	if s.storePath == "" {
		return s, nil
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// StorePath returns the persistence path (empty = memory only).
func (s *Server) StorePath() string {
	return s.storePath
}

// RegisterClient stores a client (generates ID/secret when empty).
func (s *Server) RegisterClient(c Client) *Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = "client_" + uuid.New()[:8]
	}
	if c.Secret == "" {
		c.Secret = randomToken(24)
	}
	if len(c.Scopes) == 0 {
		c.Scopes = []string{"*"}
	}
	cp := c
	s.clients[c.ID] = &cp
	_ = s.persistLocked()
	return &cp
}

// Client returns a registered client (secret redacted).
func (s *Server) Client(id string) (*Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return nil, fmt.Errorf("oauth: unknown client")
	}
	cp := *c
	cp.Secret = ""
	return &cp, nil
}

// AuthorizeParams carries authorization_code request fields including PKCE.
type AuthorizeParams struct {
	ClientID            string
	RedirectURI         string
	UserID              string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
}

// Authorize creates an authorization code (authorization_code grant with PKCE S256).
func (s *Server) Authorize(clientID, redirectURI, userID, scope string) (string, error) {
	return s.AuthorizeWith(AuthorizeParams{
		ClientID:    clientID,
		RedirectURI: redirectURI,
		UserID:      userID,
		Scope:       scope,
	})
}

// AuthorizeWith creates an authorization code using full params (PKCE required).
func (s *Server) AuthorizeWith(p AuthorizeParams) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	client, ok := s.clients[p.ClientID]
	if !ok {
		return "", fmt.Errorf("oauth: unknown client")
	}
	if !validRedirect(client.RedirectURIs, p.RedirectURI) {
		return "", fmt.Errorf("oauth: invalid redirect_uri")
	}
	method := strings.ToUpper(strings.TrimSpace(p.CodeChallengeMethod))
	if method == "" {
		method = "S256"
	}
	if method != "S256" {
		return "", fmt.Errorf("oauth: code_challenge_method must be S256")
	}
	challenge := strings.TrimSpace(p.CodeChallenge)
	if challenge == "" {
		return "", fmt.Errorf("oauth: code_challenge required")
	}
	scope := p.Scope
	if scope == "" {
		scope = strings.Join(client.Scopes, " ")
	}
	code := randomToken(32)
	s.codes[code] = authCode{
		ClientID:            p.ClientID,
		RedirectURI:         p.RedirectURI,
		UserID:              p.UserID,
		Scope:               scope,
		CodeChallenge:       challenge,
		CodeChallengeMethod: method,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}
	return code, nil
}

// Token exchanges a grant for an access token.
func (s *Server) Token(grantType, clientID, clientSecret string, params map[string]string) (*AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	client, ok := s.clients[clientID]
	if !ok {
		return nil, fmt.Errorf("oauth: invalid client credentials")
	}
	if client.Secret != "" && client.Secret != clientSecret {
		return nil, fmt.Errorf("oauth: invalid client credentials")
	}

	switch grantType {
	case "client_credentials":
		scope := ""
		if params != nil {
			scope = params["scope"]
		}
		if scope == "" {
			scope = strings.Join(client.Scopes, " ")
		}
		tok := s.issueLocked(clientID, "", scope, false)
		_ = s.persistLocked()
		return tok, nil
	case "authorization_code":
		if params == nil {
			return nil, fmt.Errorf("oauth: invalid authorization code")
		}
		code := params["code"]
		redirect := params["redirect_uri"]
		verifier := params["code_verifier"]
		entry, ok := s.codes[code]
		if !ok || time.Now().After(entry.ExpiresAt) {
			return nil, fmt.Errorf("oauth: invalid authorization code")
		}
		if entry.ClientID != clientID || entry.RedirectURI != redirect {
			return nil, fmt.Errorf("oauth: code mismatch")
		}
		if err := verifyPKCE(entry.CodeChallenge, entry.CodeChallengeMethod, verifier); err != nil {
			return nil, err
		}
		delete(s.codes, code)
		tok := s.issueLocked(clientID, entry.UserID, entry.Scope, true)
		_ = s.persistLocked()
		return tok, nil
	case "refresh_token":
		if params == nil {
			return nil, fmt.Errorf("oauth: invalid refresh_token")
		}
		plain := params["refresh_token"]
		rec, ok := s.refresh[hashToken(plain)]
		if !ok || time.Now().After(rec.ExpiresAt) {
			return nil, fmt.Errorf("oauth: invalid refresh_token")
		}
		if rec.ClientID != clientID {
			return nil, fmt.Errorf("oauth: refresh_token client mismatch")
		}
		delete(s.refresh, hashToken(plain))
		tok := s.issueLocked(clientID, rec.UserID, rec.Scope, true)
		_ = s.persistLocked()
		return tok, nil
	default:
		return nil, fmt.Errorf("oauth: unsupported grant_type")
	}
}

// Introspect validates an access token.
func (s *Server) Introspect(token string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[hashToken(token)]
	if !ok || time.Now().After(t.ExpiresAt) {
		return map[string]any{"active": false}
	}
	return map[string]any{
		"active":     true,
		"client_id":  t.ClientID,
		"user_id":    t.UserID,
		"scope":      t.Scope,
		"exp":        t.ExpiresAt.Unix(),
		"token_type": "Bearer",
	}
}

// AuthorizeHandler handles GET /oauth/authorize.
func (s *Server) AuthorizeHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		clientID := req.Query("client_id")
		redirect := req.Query("redirect_uri")
		userID := req.Query("user_id", "1")
		scope := req.Query("scope")
		state := req.Query("state")
		code, err := s.AuthorizeWith(AuthorizeParams{
			ClientID:            clientID,
			RedirectURI:         redirect,
			UserID:              userID,
			Scope:               scope,
			CodeChallenge:       req.Query("code_challenge"),
			CodeChallengeMethod: req.Query("code_challenge_method", "S256"),
		})
		if err != nil {
			return http.JSON(map[string]any{"error": "invalid_request", "error_description": err.Error()}).Status(400)
		}
		if redirect == "" {
			return http.JSON(map[string]any{"code": code, "state": state})
		}
		u, err := url.Parse(redirect)
		if err != nil {
			return http.JSON(map[string]any{"code": code, "state": state})
		}
		q := u.Query()
		q.Set("code", code)
		if state != "" {
			q.Set("state", state)
		}
		u.RawQuery = q.Encode()
		return http.Redirect(u.String(), 302)
	}
}

// TokenHandler handles POST /oauth/token.
func (s *Server) TokenHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		grant := req.Input("grant_type")
		clientID := req.Input("client_id")
		secret := req.Input("client_secret")
		if clientID == "" {
			clientID, secret, _ = parseBasicAuth(req.Header("Authorization"))
		}
		token, err := s.Token(grant, clientID, secret, map[string]string{
			"code":          req.Input("code"),
			"redirect_uri":  req.Input("redirect_uri"),
			"scope":         req.Input("scope"),
			"code_verifier": req.Input("code_verifier"),
			"refresh_token": req.Input("refresh_token"),
		})
		if err != nil {
			return http.JSON(map[string]any{"error": "invalid_grant", "error_description": err.Error()}).Status(400)
		}
		return http.JSON(token)
	}
}

// IntrospectHandler handles POST /oauth/introspect.
func (s *Server) IntrospectHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		return http.JSON(s.Introspect(req.Input("token")))
	}
}

func (s *Server) issueLocked(clientID, userID, scope string, withRefresh bool) *AccessToken {
	plain := randomToken(40)
	token := &AccessToken{
		Token:     plain,
		Type:      "Bearer",
		ExpiresIn: int(s.ttl.Seconds()),
		Scope:     scope,
		ClientID:  clientID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.ttl),
	}
	s.tokens[hashToken(plain)] = token
	if withRefresh {
		refreshPlain := randomToken(40)
		token.RefreshToken = refreshPlain
		s.refresh[hashToken(refreshPlain)] = &refreshRecord{
			Token:     refreshPlain,
			ClientID:  clientID,
			UserID:    userID,
			Scope:     scope,
			ExpiresAt: time.Now().Add(s.refreshTTL),
		}
	}
	return token
}

func (s *Server) load() error {
	raw, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	var state persistState
	if err := json.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("oauth: load store: %w", err)
	}
	if state.Clients != nil {
		s.clients = state.Clients
	}
	if state.Tokens != nil {
		s.tokens = state.Tokens
	}
	if state.Refresh != nil {
		s.refresh = state.Refresh
	}
	return nil
}

func (s *Server) persistLocked() error {
	if s.storePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o755); err != nil {
		return err
	}
	state := persistState{
		Clients: s.clients,
		Tokens:  s.tokens,
		Refresh: s.refresh,
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.storePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.storePath)
}

func verifyPKCE(challenge, method, verifier string) error {
	verifier = strings.TrimSpace(verifier)
	if verifier == "" {
		return fmt.Errorf("oauth: code_verifier required")
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method != "S256" {
		return fmt.Errorf("oauth: unsupported code_challenge_method")
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	if computed != challenge {
		return fmt.Errorf("oauth: invalid code_verifier")
	}
	return nil
}

func validRedirect(allowed []string, redirect string) bool {
	if redirect == "" {
		return true
	}
	for _, a := range allowed {
		if a == redirect || a == "*" {
			return true
		}
	}
	return false
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func parseBasicAuth(header string) (string, string, bool) {
	if !strings.HasPrefix(header, "Basic ") {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// PKCEChallengeS256 returns a code_verifier and S256 code_challenge pair for tests/clients.
func PKCEChallengeS256() (verifier, challenge string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}
