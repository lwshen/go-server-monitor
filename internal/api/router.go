package api

import (
	"io/fs"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/middleware"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
	"go.uber.org/zap"
)

// Deps is the set of dependencies the HTTP handlers need.
type Deps struct {
	Cfg   *config.Config
	Store store.Store
	Hub   *ws.Hub
	Log   *zap.Logger
	SPA   fs.FS // embedded Vue SPA (nil in dev builds; served by Vite instead)
}

// Handlers carries Deps onto the handler methods (one method per endpoint).
type Handlers struct {
	deps Deps
}

// NewRouter builds the gin engine and registers every frozen endpoint
// (REQ-RES-00). When deps.SPA is non-nil (release builds with the embedded Vue
// dist), the SPA is served as the NoRoute fallback: static files directly and
// client-side routes via index.html (see registerSPA).
func NewRouter(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(deps.Log))
	r.Use(middleware.CORS(deps.Cfg.CORSOrigins))

	h := &Handlers{deps: deps}

	// ── public ──────────────────────────────────────────
	r.GET("/health", h.Health)
	r.POST("/report", h.Report)          // probe upload; secret in body
	r.GET("/api/config", h.Config)       // public runtime config
	r.GET("/api/servers", h.Servers)     // list + stats
	r.GET("/api/server", h.ServerDetail) // one server detail (?id=)
	r.GET("/api/history", h.History)     // downsampled history (?id=&range=)
	r.GET("/ws", h.WS)                   // websocket upgrade (?subscribe=all|<id>)

	// ── admin ───────────────────────────────────────────
	r.POST("/api/admin/login", h.AdminLogin) // public: returns JWT

	admin := r.Group("/api/admin")
	admin.Use(middleware.JWTAuth(deps.Cfg.JWTSecret))
	{
		admin.POST("/servers", h.AdminServers)
		admin.POST("/servers/add", h.AdminServersAdd)
		admin.POST("/servers/edit", h.AdminServersEdit)
		admin.POST("/servers/delete", h.AdminServersDelete)
		admin.POST("/servers/reorder", h.AdminServersReorder)
		admin.GET("/settings", h.AdminGetSettings)
		admin.POST("/settings", h.AdminPostSettings)
		admin.POST("/db/rebuild", h.AdminDBRebuild)
	}

	// ── embedded SPA (release builds) ───────────────────
	if deps.SPA != nil {
		registerSPA(r, deps.SPA)
	}

	return r
}
