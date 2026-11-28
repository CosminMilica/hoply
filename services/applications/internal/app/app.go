// Package app is the composition root: it wires the repositories, services,
// routes and middleware into one http.Handler.
package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"applications/internal/config"
	"applications/internal/data/repositories"
	"applications/internal/middleware"
	"applications/internal/routes"
	"applications/internal/services"
	// FORGE:MIDDLEWARE_IMPORTS
)

// Description is served on GET /api/v1/info.
const Description = "Hoply: job search with a CV tailored to each job"

// New builds the HTTP handler. Middlewares run in list order: the first entry
// is the outermost.
func New(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
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
	items := repositories.NewPgApplicationRepository(pool)
	// FORGE:BEGIN multitenancy_rls_go:REPOSITORY_SCOPE fp:32f68c55
	items = items.WithScope(tenantScope(pool))
	// FORGE:END multitenancy_rls_go:REPOSITORY_SCOPE
	// FORGE:REPOSITORY_SCOPE
	routes.Register(router, routes.Deps{
		Pool:         pool,
		DB:           pool,
		Applications: services.NewApplicationService(items),
		Info:         routes.Info{Title: cfg.App.Name, Version: cfg.App.Version, Description: Description},
	})
	return router
}
