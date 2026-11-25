// Package claude runs the Claude Code CLI in print mode with a subscription
// token (the way work/reviewer does) and returns its structured output.
//
// The model gets no tools, no MCP servers, no session and no settings: each run
// has an empty working directory and an empty HOME, and an environment holding
// only PATH, HOME, TMPDIR, the token and DISABLE_AUTOUPDATER. The token is read
// from its file at the point of use and never logged.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Defaults of the configuration (docs/claude-setup.md).
const (
	DefaultModel    = "claude-sonnet-5-5"
	DefaultEffort   = "high"
	DefaultDeadline = 180 * time.Second
	maxPrompt       = 1 << 20
	maxOutput       = 4 << 20
)

// Config says how to run the CLI.
type Config struct {
	Binary    string
	TokenFile string
	Model     string
	Effort    string
	Deadline  time.Duration
}

// ConfigFromEnv reads CLAUDE_BIN, CLAUDE_TOKEN_FILE, CV_MODEL, CV_EFFORT and
// CLAUDE_DEADLINE (a Go duration).
func ConfigFromEnv() Config {
	cfg := Config{
		Binary:    envOr("CLAUDE_BIN", "claude"),
		TokenFile: strings.TrimSpace(os.Getenv("CLAUDE_TOKEN_FILE")),
		Model:     envOr("CV_MODEL", DefaultModel),
		Effort:    envOr("CV_EFFORT", DefaultEffort),
		Deadline:  DefaultDeadline,
	}
	if raw := strings.TrimSpace(os.Getenv("CLAUDE_DEADLINE")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			cfg.Deadline = parsed
		}
	}
	return cfg
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

// Failures of a run.
var (
	// ErrNoToken: no token file, or an empty one.
	ErrNoToken = errors.New("claude: no token installed")
	// ErrLoginNeeded: the token is invalid, expired or revoked.
	ErrLoginNeeded = errors.New("claude: login needed")
	// ErrFailed: anything else (crash, deadline, unusable answer, model error); worth one retry.
	ErrFailed = errors.New("claude: run failed")
)

// LimitError is a usage limit of the subscription, until Until.
type LimitError struct {
	Until time.Time
}

func (e *LimitError) Error() string {
	return "claude: usage limit until " + e.Until.UTC().Format(time.RFC3339)
}

// Request is one run: the system prompt, the prompt sent on stdin and the JSON
// schema of the answer.
type Request struct {
	System string
	Prompt string
	Schema string
}

// Client runs the CLI.
type Client struct {
	cfg Config
	now func() time.Time
}

func New(cfg Config) *Client {
	if cfg.Deadline <= 0 {
		cfg.Deadline = DefaultDeadline
	}
	return &Client{cfg: cfg, now: time.Now}
}

// Model is the model the client asks for.
func (c *Client) Model() string { return c.cfg.Model }

// HasToken reports whether a token is installed.
func (c *Client) HasToken() bool {
	_, err := c.token()
	return err == nil
}

func (c *Client) token() (string, error) {
	if c.cfg.TokenFile == "" {
		return "", ErrNoToken
	}
	raw, err := os.ReadFile(c.cfg.TokenFile)
	if err != nil {
		return "", ErrNoToken
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", ErrNoToken
	}
	return token, nil
}

// envelope is the part of the CLI's --output-format json result the client reads.
type envelope struct {
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	APIErrorStatus   *int            `json:"api_error_status"`
}

// Run sends the request and returns the structured output, an object that
// matches the schema.
func (c *Client) Run(ctx context.Context, req Request) (json.RawMessage, error) {
	token, err := c.token()
	if err != nil {
		return nil, err
	}
	if len(req.Prompt) > maxPrompt {
		return nil, fmt.Errorf("%w: prompt of %d bytes", ErrFailed, len(req.Prompt))
	}
	scratch, err := os.MkdirTemp("", "cv-claude-")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailed, err)
	}
	defer os.RemoveAll(scratch)
	home, work := filepath.Join(scratch, "home"), filepath.Join(scratch, "work")
	for _, dir := range []string{home, work} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFailed, err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.cfg.Deadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.cfg.Binary, c.argv(req)...)
	cmd.Dir = work
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"TMPDIR=" + scratch,
		"DISABLE_AUTOUPDATER=1",
		"CLAUDE_CODE_OAUTH_TOKEN=" + token,
	}
	cmd.Stdin = strings.NewReader(req.Prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limited{buffer: &stdout, max: maxOutput}
	cmd.Stderr = &limited{buffer: &stderr, max: 64 << 10}
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 5 * time.Second
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%w: deadline of %s passed", ErrFailed, c.cfg.Deadline)
	}

	var answer envelope
	parseErr := json.Unmarshal(stdout.Bytes(), &answer)
	if runErr != nil || answer.IsError || parseErr != nil {
		return nil, c.classify(answer, stderr.String(), runErr, parseErr)
	}
	output := bytes.TrimSpace(answer.StructuredOutput)
	if len(output) == 0 || output[0] != '{' {
		return nil, fmt.Errorf("%w: no structured output", ErrFailed)
	}
	return output, nil
}

func (c *Client) argv(req Request) []string {
	args := []string{
		"-p", "--output-format", "json",
		"--json-schema", req.Schema,
		// No built-in tools. --json-schema answers through the CLI's own StructuredOutput tool, so a
		// --disallowedTools "*" would block the answer itself (seen on 2.1.322).
		"--tools", "",
		"--strict-mcp-config",
		"--no-session-persistence",
		"--system-prompt", req.System,
	}
	if c.cfg.Model != "" {
		args = append(args, "--model", c.cfg.Model)
	}
	if c.cfg.Effort != "" {
		args = append(args, "--effort", c.cfg.Effort)
	}
	return args
}

// classify names a failed run from the CLI's own diagnostics: its result text
// and stderr, never the model's answer.
func (c *Client) classify(answer envelope, stderr string, runErr, parseErr error) error {
	detail := tail(answer.Result, 400) + " " + tail(stderr, 200)
	switch {
	case (answer.APIErrorStatus != nil && *answer.APIErrorStatus == 401) || loginDead.MatchString(detail):
		return ErrLoginNeeded
	case usageLimit.MatchString(detail):
		return &LimitError{Until: ClampReset(ParseReset(detail, c.now()), c.now())}
	case runErr != nil:
		return fmt.Errorf("%w: %v: %s", ErrFailed, runErr, strings.TrimSpace(detail))
	case parseErr != nil:
		return fmt.Errorf("%w: unreadable result: %v", ErrFailed, parseErr)
	default:
		return fmt.Errorf("%w: %s", ErrFailed, strings.TrimSpace(detail))
	}
}

func tail(text string, n int) string {
	if len(text) <= n {
		return text
	}
	return text[len(text)-n:]
}

// limited keeps at most max bytes of what is written to it.
type limited struct {
	buffer *bytes.Buffer
	max    int
}

func (l *limited) Write(p []byte) (int, error) {
	if room := l.max - l.buffer.Len(); room > 0 {
		l.buffer.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}
