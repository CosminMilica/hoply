// Package routes registers the HTTP API under /api/v1.
package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"jobs/internal/apperrors"
	"jobs/internal/services"
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
	Pool   *pgxpool.Pool
	DB     Pinger
	Jobs   *services.JobService
	Boards *services.BoardService
	Info   Info
}

// Register mounts every route on router.
func Register(router chi.Router, deps Deps) {
	router.NotFound(handle(func(_ http.ResponseWriter, r *http.Request) error {
		return apperrors.NewNotFound("Route", r.URL.Path)
	}))
	router.Route("/api/v1", func(api chi.Router) {
		registerHome(api, deps.Info)
		api.Route("/health", func(health chi.Router) { registerHealth(health, deps.DB) })
		api.Route("/jobs", func(jobs chi.Router) { registerJobs(jobs, deps.Jobs) })
		api.Route("/boards", func(boards chi.Router) { registerBoards(boards, deps.Boards) })
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

// fieldPath drops array indexes from a decoder path: "must_have.0" becomes
// "must_have".
func fieldPath(path string) string {
	parts := strings.Split(path, ".")
	kept := parts[:0]
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ".")
}

func decode(r *http.Request, target any) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return apperrors.NewValidation("request body too large")
		}
		invalid := apperrors.NewValidation("invalid JSON body: " + err.Error())
		var wrongType *json.UnmarshalTypeError
		if errors.As(err, &wrongType) && wrongType.Field != "" {
			invalid.Context = map[string]any{"field": fieldPath(wrongType.Field)}
		}
		return invalid
	}
	return nil
}
