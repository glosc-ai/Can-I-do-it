package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gloscai/template-go-vue3-docker/server/config"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(ctx context.Context, cfg config.Database) (*sql.DB, error) {
	driverName := cfg.Driver
	if driverName == "postgres" {
		driverName = "pgx"
	}

	db, err := sql.Open(driverName, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("opening %s database: %w", cfg.Driver, err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connecting to %s database: %w", cfg.Driver, err)
	}
	return db, nil
}

// Placeholder rewrites a query that uses ? as the parameter placeholder to use
// the positional $N form required by PostgreSQL. MySQL and SQLite queries are
// returned unchanged.
func Placeholder(driver, query string) string {
	if driver != "postgres" {
		return query
	}
	var b strings.Builder
	n := 0
	for _, part := range strings.Split(query, "?") {
		if n > 0 {
			fmt.Fprintf(&b, "$%d", n)
		}
		b.WriteString(part)
		n++
	}
	return b.String()
}
