package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"go.uber.org/zap/zaptest"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
)

// wsHarness spins up a router with a RUNNING hub behind a real TCP test server
// (needed for WebSocket hijacking, unlike httptest.NewRecorder).
func wsHarness(t *testing.T) (router http.Handler, base string) {
	t.Helper()
	log := zaptest.NewLogger(t)
	st, err := store.Open(context.Background(), &config.Config{DBPath: t.TempDir() + "/m.db"}, log)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	hub := ws.NewHub(log)
	go hub.Run()
	t.Cleanup(hub.Stop)

	r := NewRouter(Deps{
		Cfg:   &config.Config{APISecret: "s3cret", OfflineFactor: 5},
		Store: st,
		Hub:   hub,
		Log:   log,
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return r, srv.URL
}

func TestWebSocketBroadcast(t *testing.T) {
	r, base := wsHarness(t)
	id := register(t, r, "ws-host")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(base, "all"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 1) hello on connect
	var hello models.WsMessage
	if err := wsjson.Read(ctx, conn, &hello); err != nil {
		t.Fatalf("read hello: %v", err)
	}
	if hello.Type != "hello" || hello.Subscribed != "all" {
		t.Fatalf("hello = %+v, want type=hello subscribed=all", hello)
	}

	// 2) report -> update frame
	body := fmt.Sprintf(`{"id":%q,"secret":"s3cret","timestamp":2000000000,"data":{"cpu":55,"memory_used":100,"name":"ws-host"}}`, id)
	if code, resp := do(t, r, "POST", "/report", body); code != 200 {
		t.Fatalf("report = %d %s", code, resp)
	}

	var upd models.WsMessage
	if err := wsjson.Read(ctx, conn, &upd); err != nil {
		t.Fatalf("read update: %v", err)
	}
	if upd.Type != "update" || upd.ServerID != id {
		t.Fatalf("update = %+v, want type=update serverId=%s", upd, id)
	}
	if cpu, _ := upd.Data["cpu"].(float64); cpu != 55 {
		t.Fatalf("update cpu = %v, want 55", upd.Data["cpu"])
	}
	// static field must be excluded from the broadcast (REQ-RES-06)
	if _, ok := upd.Data["name"]; ok {
		t.Fatalf("update data leaked static field 'name': %v", upd.Data)
	}
}

func TestWebSocketScopeFilter(t *testing.T) {
	r, base := wsHarness(t)
	id := register(t, r, "ws-host")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscriber for a DIFFERENT server id must not receive this server's updates.
	conn, _, err := websocket.Dial(ctx, wsURL(base, "some-other-id"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	var hello models.WsMessage
	if err := wsjson.Read(ctx, conn, &hello); err != nil || hello.Type != "hello" {
		t.Fatalf("read hello: %v (%+v)", err, hello)
	}

	body := fmt.Sprintf(`{"id":%q,"secret":"s3cret","data":{"cpu":9}}`, id)
	if code, _ := do(t, r, "POST", "/report", body); code != 200 {
		t.Fatalf("report failed")
	}

	// Expect NO update within a short window (only a hello was due).
	rctx, rcancel := context.WithTimeout(ctx, 700*time.Millisecond)
	defer rcancel()
	var msg models.WsMessage
	if err := wsjson.Read(rctx, conn, &msg); err == nil {
		t.Fatalf("unexpected frame for out-of-scope subscriber: %+v", msg)
	}
}

func wsURL(base, scope string) string {
	return "ws" + strings.TrimPrefix(base, "http") + "/ws?subscribe=" + scope
}
