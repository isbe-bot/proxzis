package catalog

import (
	"context"
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
)

// SQLite is the local spectator archive. It is a file the daemon creates and
// seeds, so the site can be exercised without a Postgres server. The file
// holds spectator tables only.
type SQLite struct {
	*sqlStore
}

func OpenSQLite(path string) (*SQLite, error) {
	if path == "" {
		return nil, errors.New("sqlite path is empty")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&mode); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLite{sqlStore: newSQLStore(db, false)}, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

func (s *SQLite) EnsureSchema(ctx context.Context) error {
	for _, statement := range spectatorDDL {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return s.upgradeLegacyStances(ctx)
}

func (s *SQLite) upgradeLegacyStances(ctx context.Context) error {
	var legacy int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('spectator_bots') WHERE name = ?`, legacyStanceColumn).Scan(&legacy); err != nil {
		return err
	}
	if legacy == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range legacyStanceUpgrade {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}
