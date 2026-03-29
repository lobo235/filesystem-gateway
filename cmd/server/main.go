package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lobo235/filesystem-gateway/internal/api"
	"github.com/lobo235/filesystem-gateway/internal/config"
	"github.com/lobo235/filesystem-gateway/internal/nfs"
)

// version is set at build time via -ldflags "-X main.version=<value>".
var version = "dev"

func main() {
	// Bootstrap logger at INFO so we can log config errors before cfg is loaded.
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config error", "error", err)
		os.Exit(1)
	}

	// Re-create logger at the configured level.
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	log.Info("starting filesystem-gateway", "version", version, "log_level", cfg.LogLevel)

	nfsClient := nfs.NewClient(cfg.NFSBasePath, cfg.DataDir, log, cfg.MaxDownloadSize, cfg.MaxWriteFileSize, cfg.MaxExtractSize)

	// Combine built-in default download hosts with any configured additional hosts.
	downloadHosts := make([]api.AllowedHost, len(api.DefaultDownloadHosts))
	copy(downloadHosts, api.DefaultDownloadHosts)
	for _, h := range cfg.AllowedDownloadHosts {
		downloadHosts = append(downloadHosts, api.AllowedHost{
			Host:       h.Host,
			PathPrefix: h.PathPrefix,
		})
	}

	srv := api.NewServer(nfsClient, cfg.GatewayAPIKey, version, log, downloadHosts)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := ":" + cfg.Port
	if err := srv.Run(ctx, addr); err != nil {
		log.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
