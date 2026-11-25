// Package services holds the business rules. It depends on the repository
// interface, so any implementation (including in-memory test doubles) fits.
package services

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"

	"applications/internal/apperrors"
	"applications/internal/data/repositories"
	"applications/internal/identity"
	"applications/internal/models"
)

// Statuses are the accepted application statuses, in the order of the flow.
var Statuses = []string{"saved", "cv_ready", "applied"}

type ApplicationService struct {
	repo repositories.ApplicationRepository
}

func NewApplicationService(repo repositories.ApplicationRepository) *ApplicationService {
	return &ApplicationService{repo: repo}
}

func (s *ApplicationService) List(ctx context.Context, ident identity.IdentityContext) (*models.ApplicationList, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	items, err := s.repo.List(ctx, ident)
	if err != nil {
		return nil, err
	}
	return &models.ApplicationList{Items: items}, nil
}

// Put creates the application or moves it to status (docs/api/applications.md,
// "Status rules").
func (s *ApplicationService) Put(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID, data models.StatusInput) (*models.Application, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	status := strings.TrimSpace(data.Status)
	if !slices.Contains(Statuses, status) {
		return nil, invalid("status", "'status' must be one of "+strings.Join(Statuses, ", "))
	}
	return s.repo.Put(ctx, ident, jobID, status)
}

func (s *ApplicationService) Delete(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) error {
	if err := requireSubject(ident); err != nil {
		return err
	}
	removed, err := s.repo.Delete(ctx, ident, jobID)
	if err != nil {
		return err
	}
	if !removed {
		return apperrors.NewNotFound("Application", jobID.String())
	}
	return nil
}

func (s *ApplicationService) Skips(ctx context.Context, ident identity.IdentityContext) (*models.SkipList, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	ids, err := s.repo.Skips(ctx, ident)
	if err != nil {
		return nil, err
	}
	return &models.SkipList{JobIDs: ids}, nil
}

func (s *ApplicationService) Skip(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) error {
	if err := requireSubject(ident); err != nil {
		return err
	}
	return s.repo.Skip(ctx, ident, jobID)
}

func (s *ApplicationService) Unskip(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) error {
	if err := requireSubject(ident); err != nil {
		return err
	}
	return s.repo.Unskip(ctx, ident, jobID)
}

func (s *ApplicationService) ClearSkips(ctx context.Context, ident identity.IdentityContext) error {
	if err := requireSubject(ident); err != nil {
		return err
	}
	return s.repo.ClearSkips(ctx, ident)
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
