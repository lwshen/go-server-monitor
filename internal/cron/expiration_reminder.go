package cron

// CheckExpirations alerts when a server's expire_date is within 7 days
// (REQ-CRON-07).
//
// P0 STUB: logs "not implemented".
//
// TODO(P7): query servers WHERE expire_date NOT NULL AND expire_date - now <= 7d;
// for each not-yet-notified server send a reminder and set expiration_notified.
func CheckExpirations(deps Deps) {
	deps.Log.Warn("cron.CheckExpirations not implemented (P7)")
}
