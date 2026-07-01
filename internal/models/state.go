package models

// ServerState is the per-server snapshot the P7 cron jobs need: the offline
// state machine (REQ-CRON-05) and the expiration reminder (REQ-CRON-07). LastSeen
// is the newest metrics_history timestamp (Unix seconds), 0 if the server has
// never reported.
type ServerState struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Location           string `json:"location"`
	ReportInterval     int    `json:"report_interval"`
	LastOnlineState    int    `json:"last_online_state"`   // 1 online / 0 offline
	ExpireDate         string `json:"expire_date"`         // YYYY-MM-DD, "" = never
	ExpirationNotified int    `json:"expiration_notified"` // 0/1
	LastSeen           int64  `json:"last_seen"`           // Unix seconds, 0 = never
}
