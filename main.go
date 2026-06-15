package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/manager"
	"github.com/GeneralKoski/Koskidex/internal/server"
	"github.com/GeneralKoski/Koskidex/internal/storage"
)

var version = "dev"

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func main() {
	// Flags take precedence over environment variables (which act as defaults).
	port := flag.String("port", envOr("KOSKIDEX_PORT", "7700"), "HTTP port")
	dataDir := flag.String("data-dir", envOr("KOSKIDEX_DATA_DIR", "./data"), "Data directory for persistence")
	apiKey := flag.String("api-key", envOr("KOSKIDEX_API_KEY", ""), "Optional API key for authentication")
	logLevelStr := flag.String("log-level", envOr("KOSKIDEX_LOG_LEVEL", "info"), "Log level: debug, info, warn, error")
	rateLimit := flag.Int("rate-limit", envIntOr("KOSKIDEX_RATE_LIMIT", 0), "Max requests per second per IP (0 = disabled)")
	corsOrigin := flag.String("cors-origin", envOr("KOSKIDEX_CORS_ORIGIN", "*"), "Allowed CORS origin (* = any)")
	tlsCert := flag.String("tls-cert", envOr("KOSKIDEX_TLS_CERT", ""), "Path to TLS certificate file")
	tlsKey := flag.String("tls-key", envOr("KOSKIDEX_TLS_KEY", ""), "Path to TLS key file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("Koskidex version", version)
		os.Exit(0)
	}

	if (*tlsCert == "") != (*tlsKey == "") {
		fmt.Println("Both --tls-cert and --tls-key must be provided together")
		os.Exit(1)
	}

	// Parse and set log level
	var level slog.Level
	switch *logLevelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		fmt.Printf("Invalid log level %q, falling back to info\n", *logLevelStr)
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("Starting Koskidex", "port", *port, "data_dir", *dataDir)

	// Initialize manager and storage
	opts := storage.Options{DataDir: *dataDir}
	// Create manager (loaded from disk if exists)
	mgr, err := manager.NewManager(opts)
	if err != nil {
		slog.Error("Failed to initialize manager", "error", err)
		os.Exit(1)
	}
	defer mgr.Close()

	// Initialize HTTP server
	srv := server.NewServer(mgr, *apiKey, *rateLimit, *corsOrigin)
	defer srv.Close()

	addr := ":" + *port
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	shutdownErr := make(chan error, 1)
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		slog.Info("Shutdown signal received, draining connections")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		shutdownErr <- httpServer.Shutdown(ctx)
	}()

	slog.Info("HTTP server listening", "address", addr)
	var serveErr error
	if *tlsCert != "" {
		slog.Info("TLS enabled", "cert", *tlsCert)
		serveErr = httpServer.ListenAndServeTLS(*tlsCert, *tlsKey)
	} else {
		serveErr = httpServer.ListenAndServe()
	}

	if serveErr != nil && serveErr != http.ErrServerClosed {
		slog.Error("Server error", "error", serveErr)
		os.Exit(1)
	}

	if err := <-shutdownErr; err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
	}
	slog.Info("Server stopped")
}
