package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"jobs/internal/apperrors"
	"jobs/internal/models"
)

// BoardRepository keeps the company job boards people follow. Boards are
// shared: their jobs are imported for everyone.
type BoardRepository interface {
	Followed(ctx context.Context) ([]models.Board, error)
	// Follow adds a board, or keeps the one already followed; it returns the board as stored and whether it is new.
	Follow(ctx context.Context, board models.Board) (models.Board, bool, error)
}

type PgBoardRepository struct {
	pool *pgxpool.Pool
}

func NewPgBoardRepository(pool *pgxpool.Pool) *PgBoardRepository {
	return &PgBoardRepository{pool: pool}
}

func (r *PgBoardRepository) Followed(ctx context.Context) ([]models.Board, error) {
	rows, err := r.pool.Query(ctx, "SELECT platform, board, company, created_at FROM followed_boards ORDER BY created_at, platform, board")
	if err != nil {
		return nil, apperrors.FromDatabase(err)
	}
	boards, err := pgx.CollectRows(rows, scanBoard)
	if err != nil {
		return nil, apperrors.FromDatabase(err)
	}
	return nonNil(boards), nil
}

func (r *PgBoardRepository) Follow(ctx context.Context, b models.Board) (models.Board, bool, error) {
	var stored models.Board
	created := false
	err := r.pool.QueryRow(ctx, "INSERT INTO followed_boards (platform, board, company) VALUES ($1, $2, $3) "+
		"ON CONFLICT (platform, board) DO UPDATE SET platform = EXCLUDED.platform RETURNING platform, board, company, created_at, xmax = 0",
		b.Platform, b.Board, b.Company).Scan(&stored.Platform, &stored.Board, &stored.Company, &stored.FollowedAt, &created)
	if err != nil {
		return models.Board{}, false, apperrors.FromDatabase(err)
	}
	return stored, created, nil
}

func scanBoard(row pgx.CollectableRow) (models.Board, error) {
	var b models.Board
	err := row.Scan(&b.Platform, &b.Board, &b.Company, &b.FollowedAt)
	return b, err
}
