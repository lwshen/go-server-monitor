package db

import (
	"database/sql"
	"time"
)

// configurePool applies the connection-pool settings (REQ-DB-05 / REQ-DB-08.1).
//
// P0 STUB: callable but only invoked from P1 InitDB once a real *sql.DB exists.
// Values follow the spec: MaxOpenConns=25, MaxIdleConns=5, ConnMaxLifetime=0
// (SQLite needs no connection recycling).
//
// TODO(P1): call this from InitDB after opening the database.
func configurePool(db *sql.DB) {
	if db == nil {
		return
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0 * time.Second)
}
