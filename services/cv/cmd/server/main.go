// Command server runs the cv HTTP service.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cv/internal/app"
	"cv/internal/config"
	"cv/internal/db"
	// FORGE:IMPORTS
)

func main() {
	os.Exit(run())
}

func run() int {
	slog.SetDefault(app.Logger("info", false))
	// Load and validate config at startup so the fail-closed auth guard runs:
	// it rejects a production-like env with auth enabled but
	// GATEKEEPER_ISSUER/SERVICE_AUDIENCE unset.
	cfg, err := config.Load(config.LoadOptions{})
	if err != nil {
		slog.Error("configuration error", "error", err)
		return 1
	}
	slog.SetDefault(app.Logger(cfg.Logging.Level, cfg.Logging.Pretty))
	slog.Info("configuration validated", "env", cfg.App.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// FORGE:BEGIN platform_auth_go_middleware:STARTUP_INIT fp:a4407f80
	if err := app.InitPlatformAuth(cfg); err != nil {
		slog.Error("platform auth configuration error", "error", err)
		return 1
	}
	// FORGE:END platform_auth_go_middleware:STARTUP_INIT
	// FORGE:BEGIN multitenancy_rls_go:STARTUP_INIT fp:39469fb7
	if err := app.InitTenancy(); err != nil {
		slog.Error("tenancy configuration error", "error", err)
		return 1
	}
	// FORGE:END multitenancy_rls_go:STARTUP_INIT
	// FORGE:STARTUP_INIT
	// FORGE:BEGIN pii_redaction:STARTUP_INIT fp:bd51ca96
	app.InstallPIIRedaction()
	// FORGE:END pii_redaction:STARTUP_INIT

	pool, err := db.NewPool(ctx, db.URL(cfg.DB.URL), cfg.DB.PoolMax,
		time.Duration(cfg.DB.StatementTimeoutMS)*time.Millisecond)
	if err != nil {
		slog.Error("database configuration error", "error", err)
		return 1
	}
	defer pool.Close()
	// FORGE:BEGIN multitenancy_rls_go:POOL_READY fp:a420556e
	if err := app.CheckTenantRole(ctx, pool); err != nil {
		slog.Error("tenant isolation is not enforced", "error", err)
		return 1
	}
	// FORGE:END multitenancy_rls_go:POOL_READY
	// FORGE:POOL_READY

	listener, err := app.Listen(cfg.Server.Port)
	if err != nil {
		slog.Error("listen failed", "error", err)
		return 1
	}
	service, err := app.Build(cfg, pool)
	if err != nil {
		slog.Error("service configuration error", "error", err)
		return 1
	}
	go service.Worker.Run(ctx)
	if err := app.Serve(ctx, listener, service.Handler); err != nil {
		slog.Error("server failed", "error", err)
		return 1
	}
	// FORGE:SHUTDOWN
	return 0
}
