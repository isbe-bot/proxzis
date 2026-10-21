package catalog

import "database/sql"

// Postgres serves the spectator archive from the migrated database.
// OpenDatabase expects goose migrations, including 00006, to already be applied.
type Postgres struct {
	*sqlStore
}

func OpenPostgres(db *sql.DB) *Postgres {
	db.SetMaxOpenConns(8)
	return &Postgres{sqlStore: newSQLStore(db, true)}
}
