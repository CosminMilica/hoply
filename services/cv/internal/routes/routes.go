// Package routes registers the HTTP API under /api/v1.
package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cv/internal/apperrors"
	"cv/internal/services"
	// FORGE:ROUTES_IMPORTS
)

// Pinger is the readiness dependency; *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Info describes the service on GET /api/v1/info.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// Deps are the collaborators the routes use. Pool is nil in tests that do
// not touch the database.
type Deps struct {
	Pool *pgxpool.Pool
	DB   Pinger
	Cvs  *services.CvService
	Info Info
}

// Register mounts every route on router.
func Register(router chi.Router, deps Deps) {
	router.NotFound(handle(func(_ http.ResponseWriter, r *http.Request) error {
		return apperrors.NewNotFound("Route", r.URL.Path)
	}))
	router.Route("/api/v1", func(api chi.Router) {
		registerHome(api, deps.Info)
		api.Route("/health", func(health chi.Router) { registerHealth(health, deps.DB) })
		api.Route("/jobs", func(router chi.Router) { registerQuestions(router, deps.Cvs) })
		api.Route("/cvs", func(router chi.Router) { registerCvs(router, deps.Cvs) })
		// FORGE:BEGIN enhanced_health:ROUTES_REGISTRATION fp:770d7b6d
		registerDeepHealth(api)
		// FORGE:END enhanced_health:ROUTES_REGISTRATION
		// FORGE:ROUTES_REGISTRATION
	})
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// handle writes a returned error as the RFC-007 envelope.
func handle(fn handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			apperrors.Write(w, r, err)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}

func decode(r *http.Request, target any) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return apperrors.NewValidation("request body too large")
		}
		return apperrors.NewValidation("invalid JSON body: " + err.Error())
	}
	return nil
}
