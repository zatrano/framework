package orm

import (
	"database/sql"
	"reflect"

	"github.com/zatrano/framework/packages/database/query"
)

// ConnectionResolver resolves a named connection to *sql.DB and driver name.
type ConnectionResolver func(name string) (db *sql.DB, driver string, err error)

var connectionResolver ConnectionResolver

// SetConnectionResolver wires named connections (typically from database.Manager).
func SetConnectionResolver(fn ConnectionResolver) {
	connectionResolver = fn
}

// ConnectionName returns the connection name from model method Connection() string.
// Empty means the default ORM connection.
func ConnectionName[T any]() string {
	var zero T
	rv := reflect.ValueOf(&zero).Elem()
	if name := callConnectionMethod(rv); name != "" {
		return name
	}
	ptr := reflect.New(rv.Type())
	return callConnectionMethod(ptr)
}

func callConnectionMethod(rv reflect.Value) string {
	m := rv.MethodByName("Connection")
	if !m.IsValid() && rv.CanAddr() {
		m = rv.Addr().MethodByName("Connection")
	}
	if !m.IsValid() {
		return ""
	}
	results := m.Call(nil)
	if len(results) != 1 {
		return ""
	}
	return results[0].String()
}

// dbAndDriver returns the *sql.DB and driver for model T (named or default).
func dbAndDriver[T any]() (*sql.DB, string) {
	name := ConnectionName[T]()
	if name != "" && connectionResolver != nil {
		if db, driver, err := connectionResolver(name); err == nil && db != nil {
			return db, driver
		}
	}
	return DB, Driver
}

// tableQuery starts a low-level query builder on the connection for model T.
func tableQuery[T any]() *query.Builder {
	db, driver := dbAndDriver[T]()
	return query.New(db, driver, Table[T]())
}
