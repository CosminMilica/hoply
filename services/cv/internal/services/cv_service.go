// Package services holds the business rules. It depends on the repository and
// source interfaces, so any implementation (including in-memory test doubles) fits.
package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/cvdoc"
	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/internal/render"
)

// Sources reads the profile and the jobs a CV is made from.
type Sources interface {
	Profile(ctx context.Context) (models.Profile, json.RawMessage, error)
	Job(ctx context.Context, id uuid.UUID) (models.Job, json.RawMessage, error)
}

// Printer prints a page of HTML to PDF; *render.Gotenberg is one.
type Printer interface {
	PDF(ctx context.Context, page string) ([]byte, error)
}

// MaxFocus is how many skills a CV can put up front.
const MaxFocus = 3

type CvService struct {
	repo    repositories.CvRepository
	sources Sources
	printer Printer
	queued  func()
}

// NewCvService takes queued, called after a CV is queued to wake the worker.
func NewCvService(repo repositories.CvRepository, sources Sources, printer Printer, queued func()) *CvService {
	return &CvService{repo: repo, sources: sources, printer: printer, queued: queued}
}

// File is a CV as a file to download.
type File struct {
	Name        string
	ContentType string
	Body        []byte
}

// File renders a ready CV's tailored or profile version as a PDF or a DOCX.
func (s *CvService) File(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, format, version string) (*File, error) {
	if format != "pdf" && format != "docx" {
		return nil, invalid("format", "'format' must be pdf or docx")
	}
	if version == "" {
		version = "tailored"
	}
	if version != "tailored" && version != "profile" {
		return nil, invalid("version", "'version' must be tailored or profile")
	}
	cv, err := s.Get(ctx, ident, id)
	if err != nil {
		return nil, err
	}
	if cv.Status != models.StatusReady || cv.Tailored == nil || cv.Profile == nil {
		return nil, apperrors.New(apperrors.ConstraintViolation, "the CV is not ready yet")
	}
	document := *cv.Tailored
	if version == "profile" {
		document = *cv.Profile
	}
	name := strings.TrimSuffix(cv.FileName, ".pdf")
	if format == "docx" {
		body, err := render.DOCX(document)
		if err != nil {
			return nil, apperrors.NewInternal(err)
		}
		return &File{Name: name + ".docx", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", Body: body}, nil
	}
	page, err := render.HTML(document)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	pdf, err := s.printer.PDF(ctx, page)
	if err != nil {
		return nil, err
	}
	return &File{Name: name + ".pdf", ContentType: "application/pdf", Body: pdf}, nil
}

// Questions is what Prepare your CV asks for the job.
func (s *CvService) Questions(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) (*models.PrepQuestions, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	job, _, err := s.sources.Job(ctx, jobID)
	if err != nil {
		return nil, err
	}
	profile, _, err := s.sources.Profile(ctx)
	if err != nil {
		return nil, err
	}
	questions := cvdoc.Questions(cvdoc.ReadJob(job), profile)
	return &questions, nil
}

// Create snapshots the job and the profile and queues the CV.
func (s *CvService) Create(ctx context.Context, ident identity.IdentityContext, request models.CvRequest) (*models.Cv, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	jobID, err := uuid.Parse(strings.TrimSpace(request.JobID))
	if err != nil {
		return nil, invalid("job_id", "'job_id' must be a UUID")
	}
	if len(request.Focus) > MaxFocus {
		return nil, invalid("focus", "'focus' takes at most 3 skills")
	}
	for _, index := range request.Confirmed {
		if index < 0 {
			return nil, invalid("confirmed", "'confirmed' holds indexes from 0")
		}
	}
	request.JobID = jobID.String()
	request.SkipRoles = nonNil(request.SkipRoles)
	request.Focus = nonNil(request.Focus)
	if request.Confirmed == nil {
		request.Confirmed = []int{}
	}
	job, jobSnapshot, err := s.sources.Job(ctx, jobID)
	if err != nil {
		return nil, err
	}
	profile, profileSnapshot, err := s.sources.Profile(ctx)
	if err != nil {
		return nil, err
	}
	cv, err := s.repo.Create(ctx, ident, repositories.NewCv{
		JobID:           jobID,
		Request:         request,
		ProfileSnapshot: profileSnapshot,
		JobSnapshot:     jobSnapshot,
		JobTitle:        job.Title,
		Company:         job.Company,
		FileName:        cvdoc.FileName(profile.Name, job.Company),
	})
	if err != nil {
		return nil, err
	}
	s.queued()
	return cv, nil
}

func (s *CvService) Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Cv, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	cv, err := s.repo.Get(ctx, ident, id)
	if err != nil {
		return nil, err
	}
	if cv == nil {
		return nil, apperrors.NewNotFound("CV", id.String())
	}
	return cv, nil
}

// List returns the caller's CVs, newest first, for one job when jobID is set.
func (s *CvService) List(ctx context.Context, ident identity.IdentityContext, jobID string) (*models.CvList, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	var filter *uuid.UUID
	if jobID = strings.TrimSpace(jobID); jobID != "" {
		parsed, err := uuid.Parse(jobID)
		if err != nil {
			return nil, invalid("job_id", "'job_id' must be a UUID")
		}
		filter = &parsed
	}
	items, err := s.repo.List(ctx, ident, filter)
	if err != nil {
		return nil, err
	}
	return &models.CvList{Items: items}, nil
}

func (s *CvService) Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) error {
	if err := requireSubject(ident); err != nil {
		return err
	}
	removed, err := s.repo.Delete(ctx, ident, id)
	if err != nil {
		return err
	}
	if !removed {
		return apperrors.NewNotFound("CV", id.String())
	}
	return nil
}

func requireSubject(ident identity.IdentityContext) error {
	if ident.Subject == "" {
		return apperrors.New(apperrors.AuthRequired, "authentication required")
	}
	return nil
}

func invalid(field, message string) error {
	err := apperrors.NewValidation(message)
	err.Context = map[string]any{"field": field}
	return err
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
