package db

import (
	"database/sql"

	"github.com/lwshen/go-server-monitor/pkg/apperr"
)

// WithTx runs fn inside a transaction, committing on success and rolling back on
// error or panic (REQ-DB-06).
//
// P0 STUB: returns ErrNotImplemented when db is nil (the skeleton has no live DB).
// The BEGIN/COMMIT/ROLLBACK wiring below is real and ready for P1+ use.
//
// TODO(P2): used by service.SaveMetrics for batched sample inserts.
func WithTx(db *sql.DB, fn func(*sql.Tx) error) error {
	if db == nil {
		return apperr.ErrNotImplemented
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
