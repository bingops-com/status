package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bingops-com/status/internal/config"
	"github.com/bingops-com/status/internal/monitor"
	"github.com/bingops-com/status/internal/server"
	"github.com/bingops-com/status/web"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	cfg, err := config.Load(env("STATUS_CONFIG", "config/status.yaml"))
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if v := os.Getenv("STATUS_GATUS"); v != "" {
		cfg.Gatus = v
	}
	// STATUS_DEMO=true shows a made-up history; "outage" adds a service failing now.
	mode := os.Getenv("STATUS_DEMO")
	demo := mode == "true" || mode == "outage"

	opts := monitor.Options{Days: cfg.Days, Location: cfg.Location, FailureThreshold: cfg.FailureThreshold, Interval: cfg.Interval}
	if !demo {
		// The made-up history of the demo is never written next to a real one.
		opts.Path = filepath.Join(env("STATUS_DATA", "data"), "history.json")
	}
	rec, err := monitor.NewRecorder(opts)
	if err != nil {
		slog.Error("unreadable history", "error", err)
		os.Exit(1)
	}

	var src monitor.Source = &monitor.Gatus{URL: cfg.Gatus, Groups: cfg.Groups}
	if demo {
		d := monitor.Demo{Interval: cfg.Interval, Outage: mode == "outage"}
		d.Seed(rec)
		src = d
	} else if cfg.Gatus == "" {
		slog.Error("invalid configuration", "error", "gatus: set the address of the Gatus API, or STATUS_DEMO=true")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go rec.Watch(ctx, src, func(err error) { slog.Warn("reading failed", "error", err) })

	httpServer := &http.Server{
		Addr:              env("STATUS_ADDR", ":3000"),
		Handler:           (&server.Server{Config: cfg, Recorder: rec, Assets: web.Assets(), Demo: demo}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdown)
	}()

	slog.Info("status listening", "addr", httpServer.Addr, "demo", demo, "groups", cfg.Groups)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
