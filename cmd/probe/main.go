// Command probe is the monitoring agent entrypoint (REQ-PROBE-10).
//
// It wires a collect ticker, a report ticker and a 30s network-quality goroutine,
// all calling P0 stubs, plus graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/probe"
	"github.com/lwshen/go-server-monitor/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	cfg := probe.Load()
	log := logger.Init("info")
	defer func() { _ = log.Sync() }()

	collector := probe.NewCollector(cfg)
	uploader := probe.NewUploader(cfg.ServerURL, cfg.APISecret)
	buffer := probe.NewSampleBuffer(cfg.BufferSize)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	collectTicker := time.NewTicker(cfg.CollectInterval)
	defer collectTicker.Stop()
	reportTicker := time.NewTicker(cfg.ReportInterval)
	defer reportTicker.Stop()
	netTicker := time.NewTicker(30 * time.Second)
	defer netTicker.Stop()

	log.Info("探针启动",
		zap.String("server_url", cfg.ServerURL),
		zap.Duration("collect_interval", cfg.CollectInterval),
		zap.Duration("report_interval", cfg.ReportInterval),
	)

	for {
		select {
		case <-ctx.Done():
			log.Info("探针收到关闭信号，退出")
			return

		case <-collectTicker.C:
			// Collect a sample and buffer it.
			sample, err := collector.Collect()
			if err != nil {
				log.Warn("采集失败", zap.Error(err))
				continue
			}
			buffer.Append(sample)

		case <-reportTicker.C:
			// Drain the buffer and upload.
			samples := buffer.Drain()
			if len(samples) == 0 {
				continue
			}
			if err := uploader.Upload(samples); err != nil {
				// P0: Upload is a stub returning ErrNotImplemented; log and move on.
				log.Warn("上报失败 (P3 未实现)", zap.Int("samples", len(samples)), zap.Error(err))
			}

		case <-netTicker.C:
			// Probe network quality for each target (stub returns -1/-1).
			// TODO(P3): run probes concurrently and merge results into the next sample.
			for _, t := range cfg.Targets {
				_, _, _ = probe.ProbePing(t.Host, 3*time.Second)
			}
		}
	}
}
