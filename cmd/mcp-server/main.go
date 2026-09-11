package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tenqz/google-search-console-mcp/internal/config"
	"github.com/tenqz/google-search-console-mcp/internal/gsc"
	"github.com/tenqz/google-search-console-mcp/internal/httpserver"
	"github.com/tenqz/google-search-console-mcp/internal/mcpserver"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	console, err := newConsole(ctx, cfg)
	if err != nil {
		slog.Error("google search console client", "err", err)
		os.Exit(1)
	}

	mcpServer := mcpserver.New(console)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	handler := httpserver.New(cfg, mcpHandler)
	httpServer := httpserver.NewServer(cfg.Addr, handler)
	httpServer.WriteTimeout = cfg.RequestTimeout + 15*time.Second
	httpServer.BaseContext = func(net.Listener) context.Context { return ctx }

	go func() {
		slog.Info("mcp server listening",
			"addr", cfg.Addr,
			"mcpPath", cfg.MCPPath,
			"insecure", cfg.AllowInsecure,
			"demo", cfg.Demo,
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown", "err", err)
		os.Exit(1)
	}
}

// newConsole builds the live Search Console client from configured credentials.
func newConsole(ctx context.Context, cfg config.Config) (gsc.Console, error) {
	if cfg.Demo {
		return gsc.NewDemo(), nil
	}
	if cfg.GoogleCredentialsJSON != "" {
		return gsc.NewClientFromJSON(ctx, []byte(cfg.GoogleCredentialsJSON), gsc.Options{RequestTimeout: cfg.RequestTimeout, MaxConcurrent: cfg.MaxConcurrent, MaxAttempts: cfg.MaxAttempts})
	}
	return gsc.NewClientFromFile(ctx, cfg.GoogleCredentialsFile, gsc.Options{RequestTimeout: cfg.RequestTimeout, MaxConcurrent: cfg.MaxConcurrent, MaxAttempts: cfg.MaxAttempts})
}
