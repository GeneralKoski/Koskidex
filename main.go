package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/GeneralKoski/Koskidex/internal/manager"
	"github.com/GeneralKoski/Koskidex/internal/server"
	"github.com/GeneralKoski/Koskidex/internal/storage"
)

var version = "dev"

func main() {
	port := flag.String("port", "7700", "HTTP port")
	dataDir := flag.String("data-dir", "./data", "Data directory for persistence")
	apiKey := flag.String("api-key", "", "Optional API key for authentication")
	logLevelStr := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	rateLimit := flag.Int("rate-limit", 0, "Max requests per second per IP (0 = disabled)")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file")
	tlsKey := flag.String("tls-key", "", "Path to TLS key file")
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
	srv := server.NewServer(mgr, *apiKey, *rateLimit)

	// Start serving
	addr := ":" + *port
	slog.Info("HTTP server listening", "address", addr)

	if *tlsCert != "" {
		slog.Info("TLS enabled", "cert", *tlsCert)
		if err := http.ListenAndServeTLS(addr, *tlsCert, *tlsKey, srv); err != nil {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	} else {
		if err := http.ListenAndServe(addr, srv); err != nil {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}
}
