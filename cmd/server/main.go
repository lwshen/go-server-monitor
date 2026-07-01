// Command server is the single-process backend entrypoint.
//
// Startup order (REQ-PLAN-04): load .env -> config -> logger -> db -> ws hub ->
// cron -> router -> http server -> block on signal -> graceful shutdown.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/api"
	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/cron"
	"github.com/lwshen/go-server-monitor/internal/service"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
	"github.com/lwshen/go-server-monitor/pkg/logger"
)

func main() {
	// 1. Load .env (ignore if absent).
	_ = godotenv.Load()

	// 2. Load configuration.
	cfg, err := config.Load()
	if err != nil {
		// No logger yet; write to stderr and exit.
		panic(err)
	}

	// 3. Initialize the logger.
	log := logger.Init(cfg.LogLevel)
	defer func() { _ = log.Sync() }()

	// 4. Open the data store (SQLite / Turso / PostgreSQL behind one interface)
	//    and bring the schema up to date.
	st, err := store.Open(context.Background(), cfg, log)
	if err != nil {
		log.Fatal("数据库初始化失败", zap.Error(err))
	}
	defer func() { _ = st.Close() }()
	if err := st.Migrate(context.Background()); err != nil {
		log.Fatal("数据库迁移失败", zap.Error(err))
	}
	if err := service.BootstrapAdmin(context.Background(), st, cfg.AdminUsername, cfg.AdminPassword, log); err != nil {
		log.Fatal("管理员初始化失败", zap.Error(err))
	}

	// 5. Start the WebSocket hub.
	hub := ws.NewHub(log)
	go hub.Run()

	// 6. Start cron jobs.
	notifier := service.NewNotifier(st, log)
	cronScheduler, err := cron.Start(cron.Deps{
		Store:    st,
		Cfg:      cfg,
		Notifier: notifier,
		Log:      log,
	})
	if err != nil {
		log.Fatal("Cron 启动失败", zap.Error(err))
	}

	// 7. Build the HTTP router.
	router := api.NewRouter(api.Deps{
		Cfg:   cfg,
		Store: st,
		Hub:   hub,
		Log:   log,
	})

	// 8. Start the HTTP server in a goroutine.
	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: router,
	}
	go func() {
		log.Info("服务启动在 " + cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP 服务异常退出", zap.Error(err))
		}
	}()

	// 9. Block until SIGINT/SIGTERM, then shut down gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info("收到关闭信号，开始优雅关闭")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP 服务关闭超时", zap.Error(err))
	}

	hub.Stop()
	cronCtx := cronScheduler.Stop()
	<-cronCtx.Done()

	log.Info("服务已关闭")
	_ = os.Stdout.Sync()
}
