package cron

// DetectOfflineServers flips servers between online/offline and alerts on
// transitions only (REQ-CRON-05).
//
// P0 STUB: logs "not implemented".
//
// TODO(P7): for each server compare now - last sample ts against
// offline_factor × report_interval; on online->offline transition call
// notifier.NotifyOffline; persist last_online_state / last_state_change_time.
func DetectOfflineServers(deps Deps) {
	deps.Log.Warn("cron.DetectOfflineServers not implemented (P7)")
}
