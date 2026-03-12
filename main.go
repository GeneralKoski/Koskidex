package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/general-koski/koskidex/internal/server"
	"github.com/general-koski/koskidex/internal/manager"
	"github.com/general-koski/koskidex/internal/storage"
)

func main() {
	port := flag.String("port", "7700", "HTTP port")
	dataDir := flag.String("data-dir", "./data", "Data directory for persistence")
	apiKey := flag.String("api-key", "", "Optional API key for authentication")
	logLevelStr := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	flag.Parse()

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
	srv := server.NewServer(mgr, *apiKey)
	
	// Start serving
	addr := ":" + *port
	slog.Info("HTTP server listening", "address", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
