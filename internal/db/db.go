package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	goose.SetLogger(goose.NopLogger())
	goose.SetDialect("sqlite3")

	gooseProvider, err := goose.NewProvider(goose.DialectSQLite3, db, embedMigrations)
	if err != nil {
		return nil, fmt.Errorf("create goose provider: %w", err)
	}

	_, err = gooseProvider.Up(context.Background())
	if err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("database opened", "path", path)
	return &DB{DB: db}, nil
}