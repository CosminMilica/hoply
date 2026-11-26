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
	"jobs/internal/identity"
	"jobs/internal/importer"
	"jobs/internal/middleware"
	"jobs/internal/routes"
	"jobs/internal/services"
	// FORGE:MIDDLEWARE_IMPORTS
)

// Description is served on GET /api/v1/info.
const Description = "Hoply: job search with a CV tailored to each job"

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
	jobs := repositories.NewPgJobRepository(pool)
	// FORGE:REPOSITORY_SCOPE
	routes.Register(router, routes.Deps{
		Pool:   pool,
		DB:     pool,
		Jobs:   services.NewJobService(jobs, fetch.New(fetch.Options{})),
		Boards: services.NewBoardService(repositories.NewPgBoardRepository(pool), boards),
		Info:   routes.Info{Title: cfg.App.Name, Version: cfg.App.Version, Description: Description},
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
