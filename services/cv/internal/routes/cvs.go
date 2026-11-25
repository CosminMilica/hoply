package routes

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/internal/services"
)

// registerQuestions mounts /jobs/{job_id}/questions of docs/api/cv.md.
func registerQuestions(router chi.Router, cvs *services.CvService) {
	router.Get("/{job_id}/questions", handle(func(w http.ResponseWriter, r *http.Request) error {
		jobID, err := pathUUID(r, "job_id")
		if err != nil {
			return err
		}
		questions, err := cvs.Questions(r.Context(), requestIdentity(r), jobID)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, questions)
	}))
	router.Post("/{job_id}/analysis", handle(func(w http.ResponseWriter, r *http.Request) error {
		jobID, err := pathUUID(r, "job_id")
		if err != nil {
			return err
		}
		started, err := cvs.Analyze(r.Context(), requestIdentity(r), jobID)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusAccepted, map[string]string{"status": started.Status})
	}))
	router.Get("/{job_id}/probe", handle(func(w http.ResponseWriter, r *http.Request) error {
		jobID, err := pathUUID(r, "job_id")
		if err != nil {
			return err
		}
		probe, err := cvs.Probe(r.Context(), requestIdentity(r), jobID)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, probe)
	}))
}

// registerCvs mounts /cvs of docs/api/cv.md.
func registerCvs(router chi.Router, cvs *services.CvService) {
	router.Get("/", handle(func(w http.ResponseWriter, r *http.Request) error {
		list, err := cvs.List(r.Context(), requestIdentity(r), r.URL.Query().Get("job_id"))
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, list)
	}))
	router.Post("/", handle(func(w http.ResponseWriter, r *http.Request) error {
		var body models.CvRequest
		if err := decode(r, &body); err != nil {
			return err
		}
		cv, err := cvs.Create(r.Context(), requestIdentity(r), body)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusAccepted, cv)
	}))
	router.Get("/{id}", handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathUUID(r, "id")
		if err != nil {
			return err
		}
		cv, err := cvs.Get(r.Context(), requestIdentity(r), id)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, cv)
	}))
	router.Get("/{id}/file", handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathUUID(r, "id")
		if err != nil {
			return err
		}
		query := r.URL.Query()
		file, err := cvs.File(r.Context(), requestIdentity(r), id, query.Get("format"), query.Get("version"))
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", file.ContentType)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
		w.Header().Set("Content-Length", strconv.Itoa(len(file.Body)))
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(file.Body)
		return err
	}))
	router.Delete("/{id}", handle(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathUUID(r, "id")
		if err != nil {
			return err
		}
		if err := cvs.Delete(r.Context(), requestIdentity(r), id); err != nil {
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

// pathUUID parses a path parameter; anything but a UUID is 422 on that field.
func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		invalid := apperrors.NewValidation("'" + name + "' must be a UUID")
		invalid.Context = map[string]any{"field": name}
		return uuid.Nil, invalid
	}
	return id, nil
}
