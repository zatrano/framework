package database

import (
	"github.com/zatrano/framework/packages/database/query"
)

// Table starts a query builder on the default connection managed externally.
// Prefer Application.DB().Table in application code; this helper is for packages
// that already hold a manager.
func (m *Manager) Table(table string, connection ...string) (*query.Builder, error) {
	db, err := m.Connection(connection...)
	if err != nil {
		return nil, err
	}
	driver, err := m.DriverName(connection...)
	if err != nil {
		return nil, err
	}
	return query.New(db, driver, table), nil
}
