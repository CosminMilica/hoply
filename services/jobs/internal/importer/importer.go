// Package importer reads the jobs of the companies in boards.yaml from their
// public job boards, on a schedule, and keeps the ones based in Romania in the
// areas Find jobs filters on.
package importer

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"

	"jobs/internal/data/repositories"
	"jobs/internal/extract"
	"jobs/internal/fetch"
	"jobs/internal/models"
)

//go:embed boards.yaml
var boardsYAML []byte

// Board is a company's job board on one of fetch.Platforms.
type Board struct {
	Company  string `yaml:"company"`
	Platform string `yaml:"platform"`
	Board    string `yaml:"board"`
}

// Lister reads every job a board lists, and the description of a job whose
// list has none; *fetch.Reader implements it.
type Lister interface {
	List(ctx context.Context, platform, board string, country fetch.Country) ([]fetch.Posting, error)
	Describe(ctx context.Context, posting *fetch.Posting) error
}

// Romania is the market the importer keeps to.
var Romania = fetch.Country{Code: "RO", Name: "Romania"}

// Lock takes a lock only one importer holds at a time; ok is false when
// another holds it.
type Lock func(ctx context.Context) (unlock func(), ok bool, err error)

// Result is what one board's run did. Unread counts the jobs whose
// description could not be read.
type Result struct {
	Board  Board
	Listed int
	Kept   int
	New    int
	Unread int
	Closed int64
	Err    error
}

// Followed gives the boards people follow; repositories.BoardRepository implements it.
type Followed interface {
	Followed(ctx context.Context) ([]models.Board, error)
}

type Importer struct {
	store    repositories.ImportStore
	lister   Lister
	boards   []Board
	followed Followed
	lock     Lock
	pause    time.Duration
	now      func() time.Time
}

// New imports boards, then the followed ones when followed is not nil,
// through lister into store, pausing between boards.
func New(store repositories.ImportStore, lister Lister, boards []Board, followed Followed, lock Lock, pause time.Duration) *Importer {
	return &Importer{store: store, lister: lister, boards: boards, followed: followed, lock: lock, pause: pause, now: time.Now}
}

// Listed returns the boards in boards.yaml.
func (im *Importer) Listed() []Board {
	return slices.Clone(im.boards)
}

// Boards returns the boards a run reads: the listed ones, then the followed
// ones not listed. A failure to read the followed ones is logged.
func (im *Importer) Boards(ctx context.Context) []Board {
	boards := im.Listed()
	if im.followed == nil {
		return boards
	}
	followed, err := im.followed.Followed(ctx)
	if err != nil {
		slog.Error("followed job boards could not be read", "error", err)
		return boards
	}
	for _, f := range followed {
		b := Board{Company: f.Company, Platform: f.Platform, Board: f.Board}
		if !slices.ContainsFunc(boards, func(listed Board) bool { return listed.Platform == b.Platform && listed.Board == b.Board }) {
			boards = append(boards, b)
		}
	}
	return boards
}

// Import reads one board now, outside the schedule.
func (im *Importer) Import(ctx context.Context, b Board) Result {
	r := im.board(ctx, b)
	logResult(r)
	return r
}

// Interval is how often the importer runs unless JOBS_IMPORT_INTERVAL says otherwise.
const Interval = 6 * time.Hour

// FromEnv returns the importer JOBS_IMPORT=on asks for, over the boards in
// boards.yaml and the followed ones, and how often it runs; nil without
// JOBS_IMPORT=on.
func FromEnv(store repositories.ImportStore, followed Followed, pool *pgxpool.Pool) (*Importer, time.Duration, error) {
	if os.Getenv("JOBS_IMPORT") != "on" {
		return nil, 0, nil
	}
	interval := Interval
	if raw := os.Getenv("JOBS_IMPORT_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed < time.Minute {
			return nil, 0, fmt.Errorf("JOBS_IMPORT_INTERVAL must be a duration of at least 1m, got %q", raw)
		}
		interval = parsed
	}
	boards, err := ParseBoards(boardsYAML)
	if err != nil {
		return nil, 0, err
	}
	reader := fetch.New(fetch.Options{Timeout: 30 * time.Second, MaxAPIBytes: 32 << 20})
	return New(store, reader, boards, followed, PgLock(pool), time.Second), interval, nil
}

// ParseBoards reads a boards file and checks every entry.
func ParseBoards(raw []byte) ([]Board, error) {
	var file struct {
		Boards []Board `yaml:"boards"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("boards: %w", err)
	}
	seen := map[string]bool{}
	for _, b := range file.Boards {
		key := b.Platform + "/" + b.Board
		switch {
		case strings.TrimSpace(b.Company) == "" || strings.TrimSpace(b.Board) == "":
			return nil, fmt.Errorf("boards: %q needs a company and a board", key)
		case fetch.CheckBoard(b.Platform, b.Board) != nil:
			return nil, fmt.Errorf("boards: %w", fetch.CheckBoard(b.Platform, b.Board))
		case seen[key]:
			return nil, fmt.Errorf("boards: %q is listed twice", key)
		}
		seen[key] = true
	}
	return file.Boards, nil
}

// Run imports every board at once, then every interval, until ctx ends.
func (im *Importer) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		im.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce imports the boards in turn, unless another importer holds the lock.
// A board that fails is logged and skipped; the others still run.
func (im *Importer) RunOnce(ctx context.Context) []Result {
	unlock, ok, err := im.lock(ctx)
	if err != nil || !ok {
		if err != nil {
			slog.Error("job import could not take its lock", "error", err)
		}
		return nil
	}
	defer unlock()
	results := []Result{}
	for i, b := range im.Boards(ctx) {
		if i > 0 && !sleep(ctx, im.pause) {
			break
		}
		r := im.board(ctx, b)
		logResult(r)
		results = append(results, r)
	}
	return results
}

func logResult(r Result) {
	b := r.Board
	if r.Err != nil {
		slog.Warn("job board import failed", "company", b.Company, "platform", b.Platform, "error", r.Err)
		return
	}
	slog.Info("job board imported", "company", b.Company, "listed", r.Listed, "kept", r.Kept, "new", r.New, "unread", r.Unread, "closed", r.Closed)
}

// board imports one board. The description of a job whose list has none is
// read only for the jobs the list already shows as wanted. A board that lists
// no jobs, or whose descriptions could not all be read, closes nothing: an
// empty or partial answer is more likely a fault than jobs filled.
func (im *Importer) board(ctx context.Context, b Board) Result {
	r := Result{Board: b}
	postings, err := im.lister.List(ctx, b.Platform, b.Board, Romania)
	if err != nil {
		r.Err = err
		return r
	}
	r.Listed = len(postings)
	seen := im.now().UTC()
	described := 0
	for _, posting := range postings {
		if !wanted(posting) {
			continue
		}
		if described > 0 && !sleep(ctx, im.pause/4) {
			r.Err = ctx.Err()
			return r
		}
		described++
		if err := im.lister.Describe(ctx, &posting); err != nil {
			r.Unread++
			continue
		}
		job, ok := Job(b, posting)
		if !ok {
			continue
		}
		created, err := im.store.Import(ctx, job, seen)
		if err != nil {
			r.Err = err
			return r
		}
		r.Kept++
		if created {
			r.New++
		}
	}
	if r.Listed > 0 && r.Unread == 0 {
		r.Closed, r.Err = im.store.Close(ctx, b.Platform, b.Board, seen)
	}
	return r
}

// wanted reports whether a listed job may be kept, before its description is
// read: its title names an area, and it is in Romania or the platform kept the
// list to Romania.
func wanted(p fetch.Posting) bool {
	if len(extract.Areas(p.Page.Title)) == 0 {
		return false
	}
	_, country := extract.Place(strings.Join(append(slices.Clone(p.Places), p.Page.Location, p.Page.City), " · "))
	return p.InCountry || country == "Romania" || romaniaCode(p.Page.Country)
}

func romaniaCode(value string) bool {
	code := extract.Fold(strings.TrimSpace(value))
	return code == "ro" || code == "rou" || code == "romania"
}

// Job turns a posting into the job to keep, or reports false when the job is
// not based in Romania or its title names none of the areas.
func Job(b Board, p fetch.Posting) (models.ImportedJob, bool) {
	where := strings.Join(append(slices.Clone(p.Places), p.Page.Location, p.Page.City), " · ")
	city, country := extract.Place(where)
	if country == "" && (p.InCountry || romaniaCode(p.Page.Country)) {
		country = "Romania"
	}
	areas := extract.Areas(p.Page.Title)
	if country != "Romania" || len(areas) == 0 || p.ID == "" {
		return models.ImportedJob{}, false
	}
	read := extract.Read(p.Page.Text, extract.Hints{Title: p.Page.Title, Company: b.Company})
	return models.ImportedJob{
		JobInput: models.JobInput{
			Title: read.Title, Company: b.Company, City: city, Country: country,
			WorkMode: firstOf(p.Page.WorkMode, extract.WorkMode(where+" "+p.Page.Title), read.WorkMode),
			Level:    read.Level, YearsRequired: read.YearsRequired, MustHave: read.MustHave, NiceToHave: read.NiceToHave,
			Responsibilities: read.Responsibilities, About: read.About, URL: p.URL, Source: "import", ReadFrom: p.Page.ReadFrom,
		},
		Areas: areas, Origin: b.Platform, ExternalID: p.ID, Board: b.Board, PostedAt: p.PostedAt,
	}, true
}

// PgLock is a Postgres advisory lock, held on one pooled connection.
func PgLock(pool *pgxpool.Pool) Lock {
	const key = 0x686f706c79 // "hoply"
	return func(ctx context.Context) (func(), bool, error) {
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return nil, false, err
		}
		var ok bool
		if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&ok); err != nil || !ok {
			conn.Release()
			return nil, false, err
		}
		return func() {
			if _, err := conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", key); err != nil {
				// Closing the session releases its locks.
				_ = conn.Conn().Close(context.Background())
			}
			conn.Release()
		}, true, nil
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func firstOf(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
