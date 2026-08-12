package hashing

import (
	"golang.org/x/crypto/bcrypt"
)

// Manager hashes and verifies secrets.
type Manager struct {
	cost int
}

// New creates a hashing manager.
func New(cost ...int) *Manager {
	c := bcrypt.DefaultCost
	if len(cost) > 0 && cost[0] >= bcrypt.MinCost && cost[0] <= bcrypt.MaxCost {
		c = cost[0]
	}
	return &Manager{cost: c}
}

// Make creates a hash.
func (m *Manager) Make(value string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(value), m.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Check verifies a value against a hash.
func (m *Manager) Check(value, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(value)) == nil
}

// NeedsRehash reports whether a hash should be regenerated for the current cost.
func (m *Manager) NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true
	}
	return cost != m.cost
}

// Hash creates a bcrypt hash of the given value.
func Hash(value string) (string, error) {
	return New().Make(value)
}

// Check compares a plain value with a bcrypt hash.
func Check(value, hash string) bool {
	return New().Check(value, hash)
}

// NeedsRehash reports whether a hash should be regenerated.
func NeedsRehash(hash string) bool {
	return New().NeedsRehash(hash)
}
