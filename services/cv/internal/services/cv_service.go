// Package services holds the business rules. It depends on the repository and
// source interfaces, so any implementation (including in-memory test doubles) fits.
package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/cvdoc"
	"cv/internal/data/repositories"
	"cv/internal/grounding"
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

	probes repositories.Probes
	prober Prober

	letters      repositories.Letters
	letterWriter LetterWriter
	mu           sync.Mutex
	asking       map[string]bool
	settled      func()
}

// Prober asks Claude for the questions of Prepare your CV and reads job ads;
// *writer.Writer is one.
type Prober interface {
	Questions(ctx context.Context, job models.Job, profile models.Profile) ([]models.ProbeQuestion, error)
	ProfileQuestions(ctx context.Context, profile models.Profile) ([]models.ProbeQuestion, error)
	Analyze(ctx context.Context, job models.Job) error
	Available() bool
}

// Analyze starts reading the job's ad in the background, so its requirements are
// Claude's by the time the person prepares a CV. It answers pending, or
// unavailable when Claude cannot be asked.
func (s *CvService) Analyze(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) (*models.Probe, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	job, _, err := s.sources.Job(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if s.prober == nil || !s.prober.Available() {
		return &models.Probe{Status: models.ProbeUnavailable, Questions: []models.ProbeQuestion{}}, nil
	}
	key := "analysis/" + jobID.String()
	if s.startAsking(key) {
		go func() {
			defer s.doneAsking(key)
			ctx, cancel := context.WithTimeout(context.Background(), ProbeTimeout)
			defer cancel()
			if err := s.prober.Analyze(ctx, job); err != nil {
				slog.Warn("job not analyzed", "job_id", jobID, "error", err)
			}
		}()
	}
	return &models.Probe{Status: models.ProbePending, Questions: []models.ProbeQuestion{}}, nil
}

// ProbeTimeout is the longest asking for one job's questions may take: the job's analysis and two attempts, each
// up to Claude's 3-minute deadline, plus a margin.
const ProbeTimeout = 10 * time.Minute

// WithProbes lets the service ask Claude for each job's questions and keep them.
func (s *CvService) WithProbes(store repositories.Probes, prober Prober) *CvService {
	s.probes, s.prober, s.asking = store, prober, map[string]bool{}
	return s
}

// Probe is the questions Claude asks for the job: kept ones when they were asked
// for the profile's facts as they are now, else pending while they are asked for
// in the background. Unavailable when Claude cannot be asked.
func (s *CvService) Probe(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) (*models.Probe, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	if pending := s.stillAsking(ident, jobID); pending != nil {
		return pending, nil
	}
	job, _, err := s.sources.Job(ctx, jobID)
	if err != nil {
		return nil, err
	}
	profile, _, err := s.sources.Profile(ctx)
	if err != nil {
		return nil, err
	}
	fingerprint := probeFingerprint(grounding.Build(profile, cvdoc.ReadJob(job), models.CvRequest{}, time.Now()), profile)
	return s.probe(ctx, ident, jobID, fingerprint, func(ctx context.Context) ([]models.ProbeQuestion, error) {
		return s.prober.Questions(ctx, job, profile)
	})
}

// ProfileProbe is the questions Claude asks to make every CV stronger: results
// for the facts every CV leans on. They are kept like a job's, under no job.
func (s *CvService) ProfileProbe(ctx context.Context, ident identity.IdentityContext) (*models.Probe, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	if pending := s.stillAsking(ident, uuid.Nil); pending != nil {
		return pending, nil
	}
	profile, _, err := s.sources.Profile(ctx)
	if err != nil {
		return nil, err
	}
	fingerprint := probeFingerprint(grounding.Build(profile, cvdoc.Job{}, models.CvRequest{}, time.Now()), profile)
	return s.probe(ctx, ident, uuid.Nil, fingerprint, func(ctx context.Context) ([]models.ProbeQuestion, error) {
		return s.prober.ProfileQuestions(ctx, profile)
	})
}

// probeFingerprint changes when the facts or the person's language do: questions kept for other facts, or in
// another language, are asked again.
func probeFingerprint(catalogue grounding.Catalogue, profile models.Profile) string {
	return catalogue.Fingerprint() + ":" + cvdoc.LanguageOf(profile.AppLanguage)
}

func probeKey(ident identity.IdentityContext, jobID uuid.UUID) string {
	return ident.TenantID.String() + "/" + ident.Subject + "/" + jobID.String()
}

// stillAsking answers pending while the questions are being asked, before the job and the profile are read: the
// page polls every few seconds until they are ready.
func (s *CvService) stillAsking(ident identity.IdentityContext, jobID uuid.UUID) *models.Probe {
	if s.probes == nil || s.prober == nil || !s.isAsking(probeKey(ident, jobID)) {
		return nil
	}
	return &models.Probe{Status: models.ProbePending, Questions: []models.ProbeQuestion{}}
}

// probe answers the kept questions when they were asked for the facts as they
// are now, else starts asking in the background and answers pending.
func (s *CvService) probe(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID, fingerprint string,
	questions func(context.Context) ([]models.ProbeQuestion, error)) (*models.Probe, error) {
	unavailable := &models.Probe{Status: models.ProbeUnavailable, Questions: []models.ProbeQuestion{}}
	if s.probes == nil || s.prober == nil {
		return unavailable, nil
	}
	stored, err := s.probes.Probe(ctx, ident, jobID)
	if err != nil {
		return nil, err
	}
	key := probeKey(ident, jobID)
	if stored != nil && stored.Fingerprint == fingerprint {
		switch {
		case stored.Status == models.ProbeReady:
			return &models.Probe{Status: models.ProbeReady, Questions: stored.Questions}, nil
		case s.isAsking(key):
			return &models.Probe{Status: models.ProbePending, Questions: []models.ProbeQuestion{}}, nil
		case stored.Status == models.ProbeUnavailable && time.Since(stored.UpdatedAt) < ProbeTimeout:
			return unavailable, nil
		}
	}
	if !s.prober.Available() {
		return unavailable, nil
	}
	if !s.startAsking(key) {
		return &models.Probe{Status: models.ProbePending, Questions: []models.ProbeQuestion{}}, nil
	}
	pending := repositories.StoredProbe{Fingerprint: fingerprint, Status: models.ProbePending}
	if err := s.probes.SaveProbe(ctx, ident, jobID, pending); err != nil {
		s.doneAsking(key)
		return nil, err
	}
	go s.ask(ident, key, jobID, fingerprint, questions)
	return &models.Probe{Status: models.ProbePending, Questions: []models.ProbeQuestion{}}, nil
}

func (s *CvService) ask(ident identity.IdentityContext, key string, jobID uuid.UUID, fingerprint string,
	questions func(context.Context) ([]models.ProbeQuestion, error)) {
	defer s.doneAsking(key)
	ctx, cancel := context.WithTimeout(context.Background(), ProbeTimeout)
	defer cancel()
	probe := repositories.StoredProbe{Fingerprint: fingerprint, Status: models.ProbeReady}
	asked, err := questions(ctx)
	if err != nil {
		probe.Status = models.ProbeUnavailable
	}
	probe.Questions = asked
	if err := s.probes.SaveProbe(context.Background(), ident, jobID, probe); err != nil {
		slog.Error("probe not kept", "job_id", jobID, "error", err)
	}
}

func (s *CvService) isAsking(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.asking[key]
}

func (s *CvService) startAsking(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.asking[key] {
		return false
	}
	s.asking[key] = true
	return true
}

func (s *CvService) doneAsking(key string) {
	s.mu.Lock()
	delete(s.asking, key)
	settled := s.settled
	s.mu.Unlock()
	if settled != nil {
		settled()
	}
}

// OnProbeSettled calls fn each time a job's questions are kept; tests wait on it.
func (s *CvService) OnProbeSettled(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settled = fn
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

// File renders a ready CV's tailored or profile version as a PDF or a DOCX, in
// one of render.Templates.
func (s *CvService) File(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, format, version, template string) (*File, error) {
	if format != "pdf" && format != "docx" {
		return nil, invalid("format", "'format' must be pdf or docx")
	}
	if !render.Known(template) {
		return nil, invalid("template", "'template' must be one of "+strings.Join(render.Templates, ", "))
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
		body, err := render.DOCX(document, template)
		if err != nil {
			return nil, apperrors.NewInternal(err)
		}
		return &File{Name: name + ".docx", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", Body: body}, nil
	}
	page, err := render.HTML(document, template)
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
	if request.Language != "" && request.Language != cvdoc.English && request.Language != cvdoc.Romanian {
		return nil, invalid("language", "'language' must be en or ro")
	}
	request.Language = cvdoc.LanguageOf(request.Language)
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
