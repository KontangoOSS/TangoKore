package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KontangoOSS/TangoKore/internal/agent"
)

func cmdAgent(args []string) {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	identityPath := fs.String("identity", "", "path to Ziti identity JSON (default: /opt/kontango/identity.json)")
	fallbackURL := fs.String("fallback", "", "enroll API base URL for HTTPS fallback (e.g. https://your-controller.example)")
	telemetrySvc := fs.String("telemetry-service", "nats.tango", "Ziti service name for NATS telemetry bus")
	configSvc := fs.String("config-service", "config.tango", "Ziti service name for config listening")
	interval := fs.Duration("interval", 60*time.Second, "heartbeat interval")
	logLevel := fs.String("log-level", "info", "log level: debug, info, warn, error")
	fs.Parse(args)

	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	// Wrap runEnroll to match the agent.JoinFunc signature (4 params instead of 5)
	agent.JoinFunc = func(url, session, roleID, secretID string) error {
		return runEnroll(url, session, roleID, secretID, false)
	}

	cfg := agent.BootConfig{
		IdentityPath:     *identityPath,
		TelemetryService: *telemetrySvc,
		ConfigService:    *configSvc,
		FallbackURL:      *fallbackURL,
		DefaultInterval:  *interval,
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	if err := agent.Run(ctx, cfg, logger); err != nil {
		fmt.Fprintf(os.Stderr, "agent: %v\n", err)
		os.Exit(1)
	}
}
