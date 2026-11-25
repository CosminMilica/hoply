// Package app is the composition root: it wires the repositories, services,
// routes and middleware into one http.Handler.
package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"applications/internal/config"
	"applications/internal/data/repositories"
	"applications/internal/identity"
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
		// FORGE:MIDDLEWARE_REGISTRATION
		// FORGE:BEGIN rate_limit:MIDDLEWARE_REGISTRATION fp:56d180cc
		middleware.RateLimit(middleware.RateLimitOptions{RequestsPerMinute: 120}),
		// FORGE:END rate_limit:MIDDLEWARE_REGISTRATION
		// Dev-mode identity (auth.mode=none): bind the fixed local identity
		// so tenant-scoped handlers have an identity to scope on, unless an
		// outer middleware bound one. Never added when the platform-auth
		// middleware owns identity.
		devIdentity,
	}
	for _, m := range middlewares {
		router.Use(m)
	}
	applications := repositories.NewPgApplicationRepository(pool)
	// FORGE:REPOSITORY_SCOPE
	routes.Register(router, routes.Deps{
		Pool:         pool,
		DB:           pool,
		Applications: services.NewApplicationService(applications),
		Info:         routes.Info{Title: cfg.App.Name, Version: cfg.App.Version, Description: Description},
	})
	return router
}

func devIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := identity.FromContext(r.Context()); !ok {
			r = r.WithContext(identity.WithContext(r.Context(), identity.Dev()))
		}
		next.ServeHTTP(w, r)
	})
}
