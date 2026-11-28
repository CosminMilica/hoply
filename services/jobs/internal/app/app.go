// Package app is the composition root: it wires the repositories, services,
// routes and middleware into one http.Handler.
package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"jobs/internal/config"
	"jobs/internal/data/repositories"
	"jobs/internal/fetch"
	"jobs/internal/importer"
	"jobs/internal/middleware"
	"jobs/internal/routes"
	"jobs/internal/services"
	// FORGE:MIDDLEWARE_IMPORTS
)

// Description is served on GET /api/v1/info.
const Description = "crashr: job search with a CV tailored to each job"

// New builds the HTTP handler without an importer: following a board stores
// it until an importer runs (see Build).
func New(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	return newHandler(cfg, pool, nil)
}

// Service is the HTTP API and the importer JOBS_IMPORT asks for; Importer is
// nil without it, and runs every Interval once its Run is called.
type Service struct {
	Handler  http.Handler
	Importer *importer.Importer
	Interval time.Duration
}

// Build wires the API and the importer.
func Build(cfg *config.Config, pool *pgxpool.Pool) (Service, error) {
	im, interval, err := importer.FromEnv(repositories.NewPgJobRepository(pool), repositories.NewPgBoardRepository(pool), pool)
	if err != nil {
		return Service{}, err
	}
	var boards services.BoardImporter
	if im != nil {
		boards = im
	}
	return Service{Handler: newHandler(cfg, pool, boards), Importer: im, Interval: interval}, nil
}

// newHandler builds the HTTP handler. Middlewares run in list order: the
// first entry is the outermost.
func newHandler(cfg *config.Config, pool *pgxpool.Pool, boards services.BoardImporter) http.Handler {
	useErrorPort()
	router := chi.NewRouter()
	middlewares := []middleware.Middleware{
		middleware.BodyLimit(1 << 20),
		middleware.Correlation,
		middleware.Recover,
		middleware.CORS(middleware.CORSOptions{
			Enabled:          cfg.Server.CORS.Enabled,
			AllowOrigins:     cfg.Server.CORS.AllowOrigins,
			AllowCredentials: cfg.Server.CORS.AllowCredentials,
			MaxAge:           cfg.Server.CORS.MaxAge,
		}),
		// FORGE:BEGIN security_headers:MIDDLEWARE_REGISTRATION fp:444d5d13
		middleware.SecurityHeaders,
		// FORGE:END security_headers:MIDDLEWARE_REGISTRATION
		// FORGE:BEGIN platform_auth_go_middleware:MIDDLEWARE_REGISTRATION fp:fff440ab
		platformAuth(cfg),
		// FORGE:END platform_auth_go_middleware:MIDDLEWARE_REGISTRATION
		// FORGE:BEGIN multitenancy_rls_go:MIDDLEWARE_REGISTRATION fp:05e38cea
		tenantRLS(),
		// FORGE:END multitenancy_rls_go:MIDDLEWARE_REGISTRATION
		// FORGE:MIDDLEWARE_REGISTRATION
		// FORGE:BEGIN rate_limit:MIDDLEWARE_REGISTRATION fp:56d180cc
		middleware.RateLimit(middleware.RateLimitOptions{RequestsPerMinute: 120}),
		// FORGE:END rate_limit:MIDDLEWARE_REGISTRATION
	}
	for _, m := range middlewares {
		router.Use(m)
	}
	// items is the repository forge's REPOSITORY_SCOPE block binds to the caller's tenant.
	items := repositories.NewPgJobRepository(pool)
	// FORGE:BEGIN multitenancy_rls_go:REPOSITORY_SCOPE fp:32f68c55
	items = items.WithScope(tenantScope(pool))
	// FORGE:END multitenancy_rls_go:REPOSITORY_SCOPE
	// FORGE:REPOSITORY_SCOPE
	routes.Register(router, routes.Deps{
		Pool:   pool,
		DB:     pool,
		Jobs:   services.NewJobService(items, fetch.New(fetch.Options{})),
		Boards: services.NewBoardService(repositories.NewPgBoardRepository(pool), boards),
		Info:   routes.Info{Title: cfg.App.Name, Version: cfg.App.Version, Description: Description},
	})
	return router
}
