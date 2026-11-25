package routes

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"profile/internal/apperrors"
	"profile/internal/identity"
	"profile/internal/services"
)

// registerProfiles mounts the docs/api/profile.md contract.
func registerProfiles(router chi.Router, profiles *services.ProfileService) {
	router.Get("/me", handle(func(w http.ResponseWriter, r *http.Request) error {
		profile, err := profiles.Get(r.Context(), requestIdentity(r))
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, profile)
	}))
	router.Patch("/me", withBody(http.StatusOK, profiles.Patch))
	router.Put("/me/skills", withBody(http.StatusOK, profiles.SetSkills))
	router.Post("/me/experience", withBody(http.StatusCreated, profiles.AddExperience))
	router.Put("/me/experience/{id}", replacement("Experience", profiles.UpdateExperience))
	router.Delete("/me/experience/{id}", removal("Experience", profiles.DeleteExperience))
	router.Post("/me/projects", withBody(http.StatusCreated, profiles.AddProject))
	router.Put("/me/projects/{id}", replacement("Project", profiles.UpdateProject))
	router.Delete("/me/projects/{id}", removal("Project", profiles.DeleteProject))
	router.Post("/me/education", withBody(http.StatusCreated, profiles.AddEducation))
	router.Put("/me/education/{id}", replacement("Education", profiles.UpdateEducation))
	router.Delete("/me/education/{id}", removal("Education", profiles.DeleteEducation))
	router.Post("/me/certifications", withBody(http.StatusCreated, profiles.AddCertification))
	router.Put("/me/certifications/{id}", replacement("Certification", profiles.UpdateCertification))
	router.Delete("/me/certifications/{id}", removal("Certification", profiles.DeleteCertification))
	router.Post("/me/languages", withBody(http.StatusCreated, profiles.AddLanguage))
	router.Delete("/me/languages/{id}", removal("Language", profiles.DeleteLanguage))
}

// withBody decodes the JSON body, calls fn and writes its result with status.
func withBody[In, Out any](status int, fn func(context.Context, identity.IdentityContext, In) (*Out, error)) http.HandlerFunc {
	return handle(func(w http.ResponseWriter, r *http.Request) error {
		var body In
		if err := decode(r, &body); err != nil {
			return err
		}
		result, err := fn(r.Context(), requestIdentity(r), body)
		if err != nil {
			return err
		}
		return writeJSON(w, status, result)
	})
}

// replacement decodes the JSON body and replaces the {id} entity with it.
func replacement[In, Out any](entity string, fn func(context.Context, identity.IdentityContext, uuid.UUID, In) (*Out, error)) http.HandlerFunc {
	return handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, entity)
		if err != nil {
			return err
		}
		var body In
		if err := decode(r, &body); err != nil {
			return err
		}
		updated, err := fn(r.Context(), requestIdentity(r), id, body)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, updated)
	})
}

func removal(entity string, fn func(context.Context, identity.IdentityContext, uuid.UUID) error) http.HandlerFunc {
	return handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, entity)
		if err != nil {
			return err
		}
		if err := fn(r.Context(), requestIdentity(r), id); err != nil {
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

// pathID parses {id}; an id that is not a UUID cannot exist, so it is 404.
func pathID(r *http.Request, entity string) (uuid.UUID, error) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.NewNotFound(entity, raw)
	}
	return id, nil
}
