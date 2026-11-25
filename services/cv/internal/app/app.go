// Package app is the composition root: it wires the repositories, services,
// routes and middleware into one http.Handler.
package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cv/internal/config"
	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/middleware"
	"cv/internal/render"
	"cv/internal/routes"
	"cv/internal/services"
	"cv/internal/upstream"
	"cv/internal/worker"
	"cv/internal/writer"
	// FORGE:MIDDLEWARE_IMPORTS
)

// Description is served on GET /api/v1/info.
const Description = "Hoply: job search with a CV tailored to each job"

// New builds the HTTP handler without starting a worker: CVs it queues stay
// queued until a worker runs (see Build).
func New(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	return Build(cfg, pool).Handler
}

// newHandler builds the HTTP handler. Middlewares run in list order: the
// first entry is the outermost. queued is called after a CV is queued.
// Author asks Claude for the questions, the job analyses and the letters; *writer.Writer is one.
type Author interface {
	services.Prober
	services.LetterWriter
}

func newHandler(cfg *config.Config, pool *pgxpool.Pool, sources services.Sources, cvWriter routes.WriterStatus, printer services.Printer,
	queued func(), author Author) http.Handler {
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
	cvs := repositories.NewPgCvRepository(pool)
	// FORGE:REPOSITORY_SCOPE
	routes.Register(router, routes.Deps{
		Pool:   pool,
		DB:     pool,
		Cvs:    services.NewCvService(cvs, sources, printer, queued).WithProbes(cvs, author).WithLetters(cvs, author),
		Writer: cvWriter,
		Info:   routes.Info{Title: cfg.App.Name, Version: cfg.App.Version, Description: Description},
	})
	return router
}

// Service is the HTTP API and the worker that writes the queued CVs; the
// worker runs once its Run is called.
type Service struct {
	Handler http.Handler
	Worker  *worker.Worker
}

// Build wires the API and the worker, reading the profile and the jobs from
// the services PROFILE_SERVICE_URL and JOBS_SERVICE_URL name, and writing as
// CV_WRITER says (writer.FromEnv).
func Build(cfg *config.Config, pool *pgxpool.Pool) Service {
	sources := upstream.FromEnv()
	return BuildWith(cfg, pool, sources, writer.FromEnv().KeepRequirements(sources), render.GotenbergFromEnv())
}

// BuildWith wires the service on the given sources, writer and PDF printer.
func BuildWith(cfg *config.Config, pool *pgxpool.Pool, sources services.Sources, cvWriter *writer.Writer, printer services.Printer) Service {
	store := repositories.NewPgCvRepository(pool)
	queue := worker.New(store, cvWriter.KeepAnalyses(store), WorkerPoll)
	return Service{Handler: newHandler(cfg, pool, sources, cvWriter, printer, queue.Wake, cvWriter), Worker: queue}
}

// WorkerPoll is how often the worker looks at the queue when nothing woke it.
const WorkerPoll = 2 * time.Second

func devIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := identity.FromContext(r.Context()); !ok {
			r = r.WithContext(identity.WithContext(r.Context(), identity.Dev()))
		}
		next.ServeHTTP(w, r)
	})
}
