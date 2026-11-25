package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"jobs/internal/apperrors"
	"jobs/internal/identity"
	"jobs/internal/models"
	"jobs/internal/services"
)

// registerJobs mounts the docs/api/jobs.md contract.
func registerJobs(router chi.Router, jobs *services.JobService) {
	router.Get("/", handle(func(w http.ResponseWriter, r *http.Request) error {
		query := r.URL.Query()
		list, err := jobs.List(r.Context(), requestIdentity(r), services.JobQuery{
			Q: query.Get("q"), City: query.Get("city"), Country: query.Get("country"),
			IncludeRemote: query.Get("include_remote"), Areas: query["area"], Level: query.Get("level"),
		})
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, list)
	}))
	router.Post("/", handle(func(w http.ResponseWriter, r *http.Request) error {
		var body models.JobInput
		if err := decode(r, &body); err != nil {
			return err
		}
		job, err := jobs.Create(r.Context(), requestIdentity(r), body)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusCreated, job)
	}))
	router.Post("/read", handle(func(w http.ResponseWriter, r *http.Request) error {
		var body models.ReadRequest
		if err := decode(r, &body); err != nil {
			return err
		}
		draft, err := jobs.Read(r.Context(), requestIdentity(r), body)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, draft)
	}))
	router.Get("/{id}", handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		job, err := jobs.Get(r.Context(), requestIdentity(r), id)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, job)
	}))
	router.Put("/{id}/requirements", handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body models.RequirementsInput
		if err := decode(r, &body); err != nil {
			return err
		}
		job, err := jobs.SetRequirements(r.Context(), requestIdentity(r), id, body)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, job)
	}))
	router.Delete("/{id}", handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err := jobs.Delete(r.Context(), requestIdentity(r), id); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
}

// requestIdentity returns the identity the auth (or dev) middleware bound.
func requestIdentity(r *http.Request) identity.IdentityContext {
	id, _ := identity.FromContext(r.Context())
	return id
}

// pathID parses {id}; an id that is not a UUID cannot exist, so it is 404.
func pathID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.NewNotFound("Job", raw)
	}
	return id, nil
}
