package routes

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"applications/internal/apperrors"
	"applications/internal/identity"
	"applications/internal/models"
	"applications/internal/services"
)

// registerApplications mounts /applications of docs/api/applications.md.
func registerApplications(router chi.Router, applications *services.ApplicationService) {
	router.Get("/", handle(func(w http.ResponseWriter, r *http.Request) error {
		list, err := applications.List(r.Context(), requestIdentity(r))
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, list)
	}))
	router.Put("/{job_id}", handle(func(w http.ResponseWriter, r *http.Request) error {
		jobID, err := pathJobID(r)
		if err != nil {
			return err
		}
		var body models.StatusInput
		if err := decode(r, &body); err != nil {
			return err
		}
		application, err := applications.Put(r.Context(), requestIdentity(r), jobID, body)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, application)
	}))
	router.Delete("/{job_id}", noContent(applications.Delete))
}

// registerSkips mounts /skips of docs/api/applications.md.
func registerSkips(router chi.Router, applications *services.ApplicationService) {
	router.Get("/", handle(func(w http.ResponseWriter, r *http.Request) error {
		list, err := applications.Skips(r.Context(), requestIdentity(r))
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, list)
	}))
	router.Delete("/", handle(func(w http.ResponseWriter, r *http.Request) error {
		if err := applications.ClearSkips(r.Context(), requestIdentity(r)); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	router.Put("/{job_id}", noContent(applications.Skip))
	router.Delete("/{job_id}", noContent(applications.Unskip))
}

// noContent answers 204 after fn succeeds for the {job_id} in the path.
func noContent(fn func(context.Context, identity.IdentityContext, uuid.UUID) error) http.HandlerFunc {
	return handle(func(w http.ResponseWriter, r *http.Request) error {
		jobID, err := pathJobID(r)
		if err != nil {
			return err
		}
		if err := fn(r.Context(), requestIdentity(r), jobID); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// requestIdentity returns the identity the auth (or dev) middleware bound.
func requestIdentity(r *http.Request) identity.IdentityContext {
	id, _ := identity.FromContext(r.Context())
	return id
}

// pathJobID parses {job_id}; anything but a UUID is 422 on job_id.
func pathJobID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "job_id"))
	if err != nil {
		invalid := apperrors.NewValidation("'job_id' must be a UUID")
		invalid.Context = map[string]any{"field": "job_id"}
		return uuid.Nil, invalid
	}
	return id, nil
}
