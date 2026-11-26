package services

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/internal/render"
)

// LetterWriter writes a ready CV's cover letter and messages; *writer.Writer is one.
type LetterWriter interface {
	Letter(ctx context.Context, task *repositories.Task, cv models.Cv, recruiter string) models.Letter
}

const (
	// LetterTimeout is the longest writing one CV's letter may take: the job's analysis and two attempts, each up
	// to Claude's 3-minute deadline, plus a margin.
	LetterTimeout = 10 * time.Minute
	// MaxRecruiter is the longest recruiter name.
	MaxRecruiter = 80
	// MaxLetterText is the longest cover letter that can be exported.
	MaxLetterText = 8000
)

// WithLetters lets the service write each CV's cover letter and messages and keep them.
func (s *CvService) WithLetters(store repositories.Letters, author LetterWriter) *CvService {
	s.letters, s.letterWriter = store, author
	if s.asking == nil {
		s.asking = map[string]bool{}
	}
	return s
}

// Letter is the CV's cover letter and messages: none until asked for, writing,
// ready or failed. One left writing longer than LetterTimeout by a service that
// stopped is failed.
func (s *CvService) Letter(ctx context.Context, ident identity.IdentityContext, cvID uuid.UUID) (*models.Letter, error) {
	if _, err := s.Get(ctx, ident, cvID); err != nil {
		return nil, err
	}
	none := &models.Letter{Status: models.LetterNone}
	if s.letters == nil {
		return none, nil
	}
	letter, err := s.letters.Letter(ctx, ident, cvID)
	if err != nil || letter == nil {
		return none, err
	}
	if letter.Status == models.LetterWriting && !s.isAsking(letterKey(cvID)) && time.Since(letter.UpdatedAt) > LetterTimeout {
		letter.Status, letter.Reason = models.LetterFailed, models.ClaudeFailed
	}
	return letter, nil
}

// WriteLetter starts writing a ready CV's cover letter and messages in the
// background, addressed to the recruiter when one is named, and answers writing.
// Without Claude it answers failed, with the reason.
func (s *CvService) WriteLetter(ctx context.Context, ident identity.IdentityContext, cvID uuid.UUID, request models.LetterRequest) (*models.Letter, error) {
	recruiter := strings.Join(strings.Fields(request.Recruiter), " ")
	if utf8.RuneCountInString(recruiter) > MaxRecruiter {
		return nil, invalid("recruiter", "'recruiter' must be at most 80 characters")
	}
	cv, err := s.Get(ctx, ident, cvID)
	if err != nil {
		return nil, err
	}
	if cv.Status != models.StatusReady {
		return nil, apperrors.New(apperrors.ConstraintViolation, "the CV is not ready yet")
	}
	if s.letters == nil || s.letterWriter == nil {
		return &models.Letter{Status: models.LetterFailed, Recruiter: recruiter, Reason: models.RulesOff}, nil
	}
	key := letterKey(cvID)
	if s.isAsking(key) {
		return s.Letter(ctx, ident, cvID)
	}
	task, err := s.letters.Source(ctx, ident, cvID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, apperrors.NewNotFound("CV", cvID.String())
	}
	writing := models.Letter{Status: models.LetterWriting, Recruiter: recruiter}
	if err := s.letters.SaveLetter(ctx, ident, cvID, writing); err != nil {
		return nil, err
	}
	if s.startAsking(key) {
		go func() {
			defer s.doneAsking(key)
			ctx, cancel := context.WithTimeout(context.Background(), LetterTimeout)
			defer cancel()
			letter := s.letterWriter.Letter(ctx, task, *cv, recruiter)
			if err := s.letters.SaveLetter(context.Background(), ident, cvID, letter); err != nil {
				slog.Warn("letter not kept", "cv_id", cvID, "error", err)
			}
		}()
	}
	writing.UpdatedAt = time.Now()
	return &writing, nil
}

func letterKey(cvID uuid.UUID) string { return "letter/" + cvID.String() }

// LetterFile renders a cover letter, as the person edited it, as a PDF or a
// DOCX with the CV's header, in one of render.Templates.
func (s *CvService) LetterFile(ctx context.Context, ident identity.IdentityContext, cvID uuid.UUID, format, template string, body models.LetterFile) (*File, error) {
	if format != "pdf" && format != "docx" {
		return nil, invalid("format", "'format' must be pdf or docx")
	}
	if !render.Known(template) {
		return nil, invalid("template", "'template' must be one of "+strings.Join(render.Templates, ", "))
	}
	text := strings.TrimSpace(body.Text)
	if text == "" || utf8.RuneCountInString(text) > MaxLetterText {
		return nil, invalid("text", "'text' must be 1 to 8000 characters")
	}
	cv, err := s.Get(ctx, ident, cvID)
	if err != nil {
		return nil, err
	}
	if cv.Status != models.StatusReady || cv.Tailored == nil {
		return nil, apperrors.New(apperrors.ConstraintViolation, "the CV is not ready yet")
	}
	letter := render.Letter{Header: *cv.Tailored, Date: time.Now().Format("2 January 2006"), Text: text}
	name := letterFileName(cv.FileName)
	if format == "docx" {
		raw, err := render.LetterDOCX(letter, template)
		if err != nil {
			return nil, apperrors.NewInternal(err)
		}
		return &File{Name: name + ".docx", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", Body: raw}, nil
	}
	page, err := render.LetterHTML(letter, template)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	pdf, err := s.printer.PDF(ctx, page)
	if err != nil {
		return nil, err
	}
	return &File{Name: name + ".pdf", ContentType: "application/pdf", Body: pdf}, nil
}

// letterFileName is the CV's file name with "Cover_Letter" in place of "CV".
func letterFileName(cvFile string) string {
	parts := strings.Split(strings.TrimSuffix(cvFile, ".pdf"), "_")
	for i, part := range parts {
		if part == "CV" {
			parts[i] = "Cover_Letter"
			return strings.Join(parts, "_")
		}
	}
	return "Cover_Letter_" + strings.Join(parts, "_")
}
