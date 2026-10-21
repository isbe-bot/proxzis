package catalog

import (
	"context"
	"database/sql"
	"io"
	"net/url"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresURL reports whether raw is a libpq or URL connection to Postgres.
func PostgresURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "postgres", "postgresql":
		return true
	default:
		return false
	}
}

// OpenDatabase opens the spectator archive. A postgres URL uses Postgres and
// expects the goose migrations to already be applied. Any other configuration
// uses the SQLite file and creates its schema.
func OpenDatabase(ctx context.Context, databaseURL, sqlitePath string) (Store, io.Closer, error) {
	if PostgresURL(databaseURL) {
		db, err := sql.Open("pgx", databaseURL)
		if err != nil {
			return nil, nil, err
		}
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err != nil {
			_ = db.Close()
			return nil, nil, err
		}
		return OpenPostgres(db), db, nil
	}
	store, err := OpenSQLite(sqlitePath)
	if err != nil {
		return nil, nil, err
	}
	if err := store.EnsureSchema(ctx); err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	return store, store, nil
}
