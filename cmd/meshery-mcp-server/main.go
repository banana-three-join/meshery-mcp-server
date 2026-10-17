package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/meshery/meshkit/logger"
	"github.com/sirupsen/logrus"

	"github.com/meshery-extensions/meshery-mcp-server/internal/auth"
	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	mcpserver "github.com/meshery-extensions/meshery-mcp-server/internal/mcp-server"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

const appName = "meshery-mcp-server"

// Set at build time via -ldflags "-X main.version=... -X main.commitsha=...".
var (
	version   = "v0.0.0-unknown"
	commitsha = "unknown"
)

func main() {
	log, err := logger.New(appName, logger.Options{
		Format:   logger.SyslogLogFormat,
		LogLevel: int(logrus.InfoLevel),
		Output:   os.Stderr, // stdout is reserved for the MCP protocol
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := run(log); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func run(log logger.Handler) error {
	configPath := flag.String("config", "", "path to a YAML config file (optional)")
	transport := flag.String("transport", "", "stdio or http (default stdio; env MESHERY_MCP_TRANSPORT)")
	host := flag.String("host", "", "http only: loopback address to listen on (default 127.0.0.1; env MESHERY_MCP_HOST)")
	port := flag.Int("port", 0, "http only: port to listen on (default 8080; env MESHERY_MCP_PORT)")
	flag.Parse()

	// Only flags given on the command line override the environment and file.
	var flags config.Flags
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "transport":
			flags.Transport = transport
		case "host":
			flags.Host = host
		case "port":
			flags.Port = port
		}
	})

	cfg, err := config.Load(*configPath, flags)
	if err != nil {
		return err
	}
	log.SetLevel(logrusLevel(cfg.Log.Level))

	log.Infof("starting %s version=%s commit=%s meshery_url=%s token_configured=%t transport=%s",
		appName, version, commitsha, cfg.Meshery.RedactedURL(), cfg.Auth.Token != "", cfg.Transport.Mode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		// After the first signal, restore default handling so a second
		// Ctrl+C exits immediately instead of waiting for shutdown.
		<-ctx.Done()
		stop()
	}()

	opts := mcpserver.Options{
		Name:        appName,
		Version:     version,
		Logger:      slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel(cfg.Log.Level)})),
		Registry:    meshery.NewRegistry(meshery.NewClient(cfg.Meshery.URL)),
		Credentials: auth.Resolver(cfg.Auth),
	}

	switch cfg.Transport.Mode {
	case config.TransportHTTP:
		err = mcpserver.RunHTTP(ctx, opts, mcpserver.HTTPOptions{
			Addr:      net.JoinHostPort(cfg.Transport.Host, strconv.Itoa(cfg.Transport.Port)),
			AccessKey: string(cfg.Transport.AccessKey),
		})
	default:
		err = mcpserver.RunStdio(ctx, opts)
	}
	if err != nil {
		return err
	}
	log.Info("stopped")
	return nil
}

func logrusLevel(level string) logrus.Level {
	switch level {
	case config.LogLevelDebug:
		return logrus.DebugLevel
	case config.LogLevelWarn:
		return logrus.WarnLevel
	case config.LogLevelError:
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

func slogLevel(level string) slog.Level {
	switch level {
	case config.LogLevelDebug:
		return slog.LevelDebug
	case config.LogLevelWarn:
		return slog.LevelWarn
	case config.LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
