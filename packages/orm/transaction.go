package orm

import (
	"database/sql"
	"fmt"

	"github.com/zatrano/framework/packages/database/query"
)

// Transaction runs fn inside a database transaction.
// Use QueryTx / query.New(tx, ...) inside fn so ORM operations participate.
// On success the transaction is committed; on error it is rolled back.
func Transaction(fn func(tx *sql.Tx) error) (err error) {
	if DB == nil {
		return fmt.Errorf("orm database is not configured")
	}
	if fn == nil {
		return fmt.Errorf("transaction callback is nil")
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	err = fn(tx)
	return err
}

// QueryTx starts a model query bound to a transaction.
func QueryTx[T any](tx *sql.Tx) *Querier[T] {
	return QueryOn[T](tx)
}

// QueryOn starts a model query on any DBTX (*sql.DB or *sql.Tx).
// Driver name comes from the model's Connection() when resolvable, else the default.
func QueryOn[T any](db query.DBTX) *Querier[T] {
	table := Table[T]()
	_, driver := dbAndDriver[T]()
	return &Querier[T]{
		builder:    query.New(db, driver, table),
		table:      table,
		softDelete: hasSoftDeletes[T](),
	}
}

// TransactionOn runs fn in a transaction on a named connection (empty = default).
func TransactionOn(connection string, fn func(tx *sql.Tx) error) (err error) {
	var db *sql.DB
	if connection != "" && connectionResolver != nil {
		var resolveErr error
		db, _, resolveErr = connectionResolver(connection)
		if resolveErr != nil {
			return resolveErr
		}
	} else {
		db = DB
	}
	if db == nil {
		return fmt.Errorf("orm database is not configured")
	}
	if fn == nil {
		return fmt.Errorf("transaction callback is nil")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	err = fn(tx)
	return err
}
