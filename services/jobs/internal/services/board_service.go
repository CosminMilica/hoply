package services

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"jobs/internal/apperrors"
	"jobs/internal/data/repositories"
	"jobs/internal/fetch"
	"jobs/internal/identity"
	"jobs/internal/importer"
	"jobs/internal/models"
)

// BoardImporter imports company job boards; *importer.Importer implements it.
type BoardImporter interface {
	Listed() []importer.Board
	Import(ctx context.Context, b importer.Board) importer.Result
}

// FollowTimeout bounds the import a follow starts in the background.
const FollowTimeout = 10 * time.Minute

type BoardService struct {
	repo     repositories.BoardRepository
	importer BoardImporter
	// background runs the import a follow starts; tests run it inline.
	background func(func())
}

// NewBoardService follows boards in repo; with an importer, a followed board is
// imported at once.
func NewBoardService(repo repositories.BoardRepository, importer BoardImporter) *BoardService {
	return &BoardService{repo: repo, importer: importer, background: func(run func()) { go run() }}
}

// WithBackground replaces how a follow's import runs.
func (s *BoardService) WithBackground(background func(func())) *BoardService {
	s.background = background
	return s
}

// Of returns the board a job link belongs to; 404 when crashr cannot read it.
func (s *BoardService) Of(ctx context.Context, ident identity.IdentityContext, link string) (*models.BoardOf, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	platform, board, ok := fetch.BoardOf(link)
	if !ok {
		return nil, apperrors.NewNotFound("Board", link)
	}
	of := &models.BoardOf{Platform: platform, Board: board}
	known, err := s.known(ctx)
	if err != nil {
		return nil, err
	}
	if i := slices.IndexFunc(known, func(b models.Board) bool { return b.Platform == platform && b.Board == board }); i >= 0 {
		of.Company, of.Imported = known[i].Company, true
	}
	return of, nil
}

// Follow follows the board of a job link and, with an importer, imports a newly
// followed board in the background. A board already listed or followed is
// returned as it is.
func (s *BoardService) Follow(ctx context.Context, ident identity.IdentityContext, in models.FollowInput) (*models.Board, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	company := strings.TrimSpace(in.Company)
	if company == "" || utf8.RuneCountInString(company) > maxLocation {
		return nil, invalid("company", "'company' is required, at most 120 characters")
	}
	platform, board, ok := fetch.BoardOf(in.URL)
	if !ok {
		return nil, invalid("url", "'url' must be a job link on a board crashr reads: Greenhouse, Lever, Ashby, SmartRecruiters, Workday, BambooHR, Recruitee or Workable")
	}
	if s.importer != nil {
		for _, listed := range s.importer.Listed() {
			if listed.Platform == platform && listed.Board == board {
				return &models.Board{Company: listed.Company, Platform: platform, Board: board}, nil
			}
		}
	}
	stored, created, err := s.repo.Follow(ctx, models.Board{Company: company, Platform: platform, Board: board})
	if err != nil {
		return nil, err
	}
	// A board followed before is already imported on every run; only a new one is imported at once.
	if created && s.importer != nil {
		follow := importer.Board{Company: stored.Company, Platform: stored.Platform, Board: stored.Board}
		s.background(func() {
			ctx, cancel := context.WithTimeout(context.Background(), FollowTimeout)
			defer cancel()
			if r := s.importer.Import(ctx, follow); r.Err != nil {
				slog.Warn("a followed job board could not be imported yet", "company", follow.Company, "error", r.Err)
			}
		})
	}
	return &stored, nil
}

// known returns the boards whose jobs are imported: the listed ones and the
// followed ones.
func (s *BoardService) known(ctx context.Context) ([]models.Board, error) {
	known := []models.Board{}
	if s.importer != nil {
		for _, b := range s.importer.Listed() {
			known = append(known, models.Board{Company: b.Company, Platform: b.Platform, Board: b.Board})
		}
	}
	followed, err := s.repo.Followed(ctx)
	if err != nil {
		return nil, err
	}
	return append(known, followed...), nil
}
