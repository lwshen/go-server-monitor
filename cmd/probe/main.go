// Command probe is the monitoring agent entrypoint (REQ-PROBE-10).
//
// It wires a collect ticker, a report ticker and a 30s network-quality probe, plus
// graceful shutdown on SIGINT/SIGTERM. Collect and report intervals are separate:
// when collect < report, one upload carries several buffered samples (REQ-RES-04).
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

	if cfg.ServerURL == "" || cfg.ServerID == "" || cfg.APISecret == "" {
		log.Fatal("探针配置缺失：SERVER_URL / SERVER_ID / API_SECRET 均为必填",
			zap.Bool("has_server_url", cfg.ServerURL != ""),
			zap.Bool("has_server_id", cfg.ServerID != ""),
			zap.Bool("has_secret", cfg.APISecret != ""))
	}

	collector := probe.NewCollector(cfg)
	uploader := probe.NewUploader(cfg)
	buffer := probe.NewSampleBuffer(cfg.BufferSize)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("探针启动",
		zap.String("server_url", cfg.ServerURL),
		zap.String("server_id", cfg.ServerID),
		zap.Duration("collect_interval", cfg.CollectInterval),
		zap.Duration("report_interval", cfg.ReportInterval),
	)

	// Kick off an initial network probe (may take a few seconds) and collect one
	// sample immediately so the first scheduled report is not empty.
	go collector.ProbeNetwork()
	collectOnce(collector, buffer, log)

	collectTicker := time.NewTicker(cfg.CollectInterval)
	defer collectTicker.Stop()
	reportTicker := time.NewTicker(cfg.ReportInterval)
	defer reportTicker.Stop()
	netTicker := time.NewTicker(30 * time.Second)
	defer netTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("探针收到关闭信号，最后上报一次并退出")
			flush(uploader, buffer, log)
			return

		case <-collectTicker.C:
			collectOnce(collector, buffer, log)

		case <-reportTicker.C:
			flush(uploader, buffer, log)

		case <-netTicker.C:
			go collector.ProbeNetwork()
		}
	}
}

// collectOnce samples the host and buffers the result.
func collectOnce(c *probe.Collector, buf *probe.SampleBuffer, log *zap.Logger) {
	sample, err := c.Collect()
	if err != nil {
		log.Warn("采集失败", zap.Error(err))
		return
	}
	buf.Append(sample)
	log.Debug("采集成功",
		zap.Float64("cpu", sample.Cpu),
		zap.Float64("mem_used_mib", sample.MemoryUsed),
		zap.Int("buffered", buf.Len()))
}

// flush drains the buffer and uploads; on permanent failure the samples are
// already gone (dropped) to avoid unbounded growth against a misconfigured server.
func flush(u *probe.Uploader, buf *probe.SampleBuffer, log *zap.Logger) {
	samples := buf.Drain()
	if len(samples) == 0 {
		return
	}
	if err := u.Upload(samples); err != nil {
		log.Warn("上报失败", zap.Int("samples", len(samples)), zap.Error(err))
		return
	}
	log.Info("上报成功", zap.Int("samples", len(samples)))
}
