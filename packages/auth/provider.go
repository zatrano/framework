package auth

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/hashing"
)

// GenericUser is a map-backed authenticatable user.
type GenericUser struct {
	Attributes map[string]any
}

// AuthID returns the user id.
func (u *GenericUser) AuthID() any {
	if u == nil {
		return nil
	}
	if id, ok := u.Attributes["id"]; ok {
		return id
	}
	return nil
}

// AuthPassword returns the password hash.
func (u *GenericUser) AuthPassword() string {
	if u == nil {
		return ""
	}
	if password, ok := u.Attributes["password"].(string); ok {
		return password
	}
	return fmt.Sprint(u.Attributes["password"])
}

// Get returns an attribute.
func (u *GenericUser) Get(key string) any {
	return u.Attributes[key]
}

// GetEmailForPasswordReset returns the email used for password resets.
func (u *GenericUser) GetEmailForPasswordReset() string {
	if u == nil {
		return ""
	}
	return fmt.Sprint(u.Get("email"))
}

// DatabaseUserProvider retrieves users from a database table.
type DatabaseUserProvider struct {
	db         *sql.DB
	driver     string
	table      string
	idColumn   string
	passColumn string
	// Hydrate maps a DB row to an Authenticatable (e.g. *models.User). When nil, GenericUser is used.
	Hydrate func(row map[string]any) Authenticatable
}

// NewDatabaseUserProvider creates a database user provider.
func NewDatabaseUserProvider(db *sql.DB, driver, table string) *DatabaseUserProvider {
	return &DatabaseUserProvider{
		db:         db,
		driver:     driver,
		table:      table,
		idColumn:   "id",
		passColumn: "password",
	}
}

// WithHydrate sets a row→user mapper and returns the provider.
func (p *DatabaseUserProvider) WithHydrate(fn func(row map[string]any) Authenticatable) *DatabaseUserProvider {
	p.Hydrate = fn
	return p
}

func (p *DatabaseUserProvider) hydrate(row map[string]any) Authenticatable {
	if row == nil {
		return nil
	}
	if p.Hydrate != nil {
		return p.Hydrate(row)
	}
	return &GenericUser{Attributes: row}
}

// RetrieveByID finds a user by id.
func (p *DatabaseUserProvider) RetrieveByID(id any) (Authenticatable, error) {
	row, err := query.New(p.db, p.driver, p.table).Where(p.idColumn, id).First()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p.hydrate(row), nil
}

// RetrieveByCredentials finds a user by login credentials (excluding password).
func (p *DatabaseUserProvider) RetrieveByCredentials(credentials map[string]string) (Authenticatable, error) {
	builder := query.New(p.db, p.driver, p.table)
	found := false
	for key, value := range credentials {
		if key == p.passColumn || key == "password" {
			continue
		}
		builder.Where(key, value)
		found = true
	}
	if !found {
		return nil, fmt.Errorf("credentials require a non-password field")
	}
	row, err := builder.First()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p.hydrate(row), nil
}

// ValidateCredentials validates the password.
func (p *DatabaseUserProvider) ValidateCredentials(user Authenticatable, credentials map[string]string) bool {
	password := credentials["password"]
	if password == "" {
		return false
	}
	return hashing.Check(password, user.AuthPassword())
}

// UpdatePassword updates a user's password by email.
func (p *DatabaseUserProvider) UpdatePassword(email, hashedPassword string) error {
	_, err := query.New(p.db, p.driver, p.table).
		Where("email", email).
		Update(map[string]any{"password": hashedPassword})
	return err
}

// UpdateAttributes updates columns for a user id.
func (p *DatabaseUserProvider) UpdateAttributes(id any, attrs map[string]any) error {
	_, err := query.New(p.db, p.driver, p.table).
		Where(p.idColumn, id).
		Update(attrs)
	return err
}

// RetrieveByToken finds a user by id + remember token.
func (p *DatabaseUserProvider) RetrieveByToken(id, token string) (Authenticatable, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	row, err := query.New(p.db, p.driver, p.table).
		Where(p.idColumn, id).
		Where("remember_token", token).
		First()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &GenericUser{Attributes: row}, nil
}

// UpdateRememberToken stores (or clears) the remember token for a user.
func (p *DatabaseUserProvider) UpdateRememberToken(user Authenticatable, token string) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}
	attrs := map[string]any{"remember_token": nil}
	if strings.TrimSpace(token) != "" {
		attrs["remember_token"] = token
	}
	return p.UpdateAttributes(user.AuthID(), attrs)
}

// Create inserts a user row and returns the authenticatable record.
func (p *DatabaseUserProvider) Create(attrs map[string]any) (Authenticatable, error) {
	if len(attrs) == 0 {
		return nil, fmt.Errorf("attributes required")
	}
	id, err := query.New(p.db, p.driver, p.table).InsertGetID(attrs)
	if err != nil {
		return nil, err
	}
	return p.RetrieveByID(id)
}
