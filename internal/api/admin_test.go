package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/service"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
)

// adminHarness builds a router with a configured JWT secret + a bootstrapped admin
// (admin/pw123), and returns a valid token for authenticated calls.
func adminHarness(t *testing.T) (http.Handler, store.Store, string) {
	t.Helper()
	ctx := context.Background()
	log := zaptest.NewLogger(t)
	st, err := store.Open(ctx, &config.Config{DBPath: filepath.Join(t.TempDir(), "m.db")}, log)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := service.BootstrapAdmin(ctx, st, "admin", "pw123", log); err != nil {
		t.Fatalf("BootstrapAdmin: %v", err)
	}
	r := NewRouter(Deps{
		Cfg:   &config.Config{APISecret: "s3cret", JWTSecret: "testjwt", OfflineFactor: 5},
		Store: st,
		Hub:   ws.NewHub(log),
		Log:   log,
	})
	token, err := service.IssueJWT("testjwt", "admin")
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}
	return r, st, token
}

func authReq(t *testing.T, h http.Handler, method, path, body, token string) (int, string) {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func TestAdminLogin(t *testing.T) {
	h, _, _ := adminHarness(t)

	code, body := do(t, h, http.MethodPost, "/api/admin/login", `{"username":"admin","password":"pw123"}`)
	if code != 200 {
		t.Fatalf("login = %d %s, want 200", code, body)
	}
	var out struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}
	if json.Unmarshal([]byte(body), &out); out.Token == "" || out.ExpiresIn != 604800 {
		t.Fatalf("login body = %s, want token + expires_in 604800", body)
	}
	// The issued token must authenticate an admin request.
	if c, _ := authReq(t, h, http.MethodPost, "/api/admin/servers", "", out.Token); c != 200 {
		t.Fatalf("issued token rejected: %d", c)
	}

	if c, _ := do(t, h, http.MethodPost, "/api/admin/login", `{"username":"admin","password":"WRONG"}`); c != 401 {
		t.Fatalf("wrong password = %d, want 401", c)
	}
}

func TestAdminAuthRequired(t *testing.T) {
	h, _, token := adminHarness(t)
	if c, _ := authReq(t, h, http.MethodPost, "/api/admin/servers", "", ""); c != 401 {
		t.Fatalf("no token = %d, want 401", c)
	}
	if c, _ := authReq(t, h, http.MethodPost, "/api/admin/servers", "", "garbage.token"); c != 401 {
		t.Fatalf("bad token = %d, want 401", c)
	}
	if c, _ := authReq(t, h, http.MethodPost, "/api/admin/servers", "", token); c != 200 {
		t.Fatalf("valid token = %d, want 200", c)
	}
}

func TestAdminServerCRUD(t *testing.T) {
	h, _, token := adminHarness(t)

	// add
	code, body := authReq(t, h, http.MethodPost, "/api/admin/servers/add", `{"name":"srv"}`, token)
	if code != 200 {
		t.Fatalf("add = %d %s", code, body)
	}
	var added struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(body), &added)
	if added.ID == "" {
		t.Fatalf("add returned no id: %s", body)
	}

	// edit (partial: only name)
	if c, b := authReq(t, h, http.MethodPost, "/api/admin/servers/edit", `{"id":"`+added.ID+`","name":"renamed","server_group":"grp"}`, token); c != 200 {
		t.Fatalf("edit = %d %s", c, b)
	}

	// list reflects the edit
	_, listBody := authReq(t, h, http.MethodPost, "/api/admin/servers", "", token)
	if !strings.Contains(listBody, `"name":"renamed"`) || !strings.Contains(listBody, `"server_group":"grp"`) {
		t.Fatalf("list after edit = %s, want renamed/grp", listBody)
	}

	// delete
	if c, b := authReq(t, h, http.MethodPost, "/api/admin/servers/delete", `{"id":"`+added.ID+`"}`, token); c != 200 {
		t.Fatalf("delete = %d %s", c, b)
	}
	_, listBody = authReq(t, h, http.MethodPost, "/api/admin/servers", "", token)
	if strings.Contains(listBody, added.ID) {
		t.Fatalf("server still present after delete: %s", listBody)
	}
	// deleting a missing id -> 404
	if c, _ := authReq(t, h, http.MethodPost, "/api/admin/servers/delete", `{"id":"`+added.ID+`"}`, token); c != 404 {
		t.Fatalf("delete missing = %d, want 404", c)
	}
}

func TestAdminSettings(t *testing.T) {
	h, st, token := adminHarness(t)
	ctx := context.Background()
	_ = st.SetSetting(ctx, "telegram_bot_token", "sekret") // a secret-class key

	// POST: site_title is written; api_secret (write-protected) is skipped.
	if c, b := authReq(t, h, http.MethodPost, "/api/admin/settings", `{"site_title":"My Mon","api_secret":"hack"}`, token); c != 200 {
		t.Fatalf("post settings = %d %s", c, b)
	}

	_, body := authReq(t, h, http.MethodGet, "/api/admin/settings", "", token)
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("settings json: %v", err)
	}
	if m["site_title"] != "My Mon" {
		t.Fatalf("site_title = %v, want My Mon", m["site_title"])
	}
	if _, leaked := m["telegram_bot_token"]; leaked {
		t.Fatalf("secret telegram_bot_token leaked in plaintext: %s", body)
	}
	if m["telegram_bot_token_set"] != true {
		t.Fatalf("telegram_bot_token_set = %v, want true", m["telegram_bot_token_set"])
	}
	// api_secret must not have been written via the settings endpoint.
	if v, _ := st.GetSetting(ctx, "api_secret"); v != "" {
		t.Fatalf("write-protected api_secret was set to %q", v)
	}
}

func TestAdminDBRebuild(t *testing.T) {
	h, _, token := adminHarness(t)
	if c, _ := authReq(t, h, http.MethodPost, "/api/admin/db/rebuild", `{}`, token); c != 400 {
		t.Fatalf("rebuild without confirm = %d, want 400", c)
	}
	if c, b := authReq(t, h, http.MethodPost, "/api/admin/db/rebuild", `{"confirm":true}`, token); c != 200 {
		t.Fatalf("rebuild confirmed = %d %s, want 200", c, b)
	}
}
