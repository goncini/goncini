package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

//go:embed schema.sql
var schema string

// sqliteOptions are applied to every pooled connection. Immediate
// transactions take the write lock up front, so concurrent writers wait on
// busy_timeout instead of failing to upgrade a read lock.
const sqliteOptions = "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"

func openDB(ctx context.Context, dsn string) (*sql.DB, error) {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := sql.Open("sqlite", dsn+sep+sqliteOptions)
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// migrate applies the Up part of schema.sql to a new database, tracking it
// with SQLite's user_version.
func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var version int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version >= 1 {
		return nil
	}
	up, _, _ := strings.Cut(schema, "-- +goose Down")
	if _, err := tx.ExecContext(ctx, up); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
		return err
	}
	return tx.Commit()
}

// isUniqueViolation reports whether err is a UNIQUE constraint failure on
// column, given as "table.column".
func isUniqueViolation(err error, column string) bool {
	var e *sqlite.Error
	return errors.As(err, &e) &&
		e.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE &&
		strings.Contains(e.Error(), column)
}

// Times are stored as Unix microseconds.

func nowMicros() int64 { return time.Now().UnixMicro() }

func fromMicros(us int64) time.Time { return time.UnixMicro(us).UTC() }
