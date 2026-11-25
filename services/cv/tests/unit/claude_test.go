package unit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cv/internal/claude"
)

// fakeClaude writes a script that plays the CLI: it records its argv, its
// environment and its stdin next to itself, prints the envelope file and exits
// with code. It returns the script and the directory of the records.
func fakeClaude(t *testing.T, envelope string, code int) (string, string) {
	t.Helper()
	dir := t.TempDir()
	body := "#!/bin/sh\n" +
		"for arg in \"$@\"; do printf '%s\\n' \"$arg\"; done > '" + dir + "/argv'\n" +
		"env > '" + dir + "/env'\n" +
		"pwd > '" + dir + "/pwd'\n" +
		"cat > '" + dir + "/stdin'\n"
	if envelope != "" {
		body += "cat '" + envelope + "'\n"
	}
	body += "exit " + string(rune('0'+code)) + "\n"
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script, dir
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs("../testdata/claude/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func tokenFile(t *testing.T, token string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude_oauth_token")
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

var sampleRequest = claude.Request{System: "You write CVs.", Prompt: `{"facts": []}`, Schema: `{"type":"object"}`}

func TestClaudeRunsTheCLIIsolatedAndReturnsTheStructuredOutput(t *testing.T) {
	script, records := fakeClaude(t, fixture(t, "success.synthetic.json"), 0)
	t.Setenv("ANTHROPIC_API_KEY", "must-not-leak")
	client := claude.New(claude.Config{Binary: script, TokenFile: tokenFile(t, " sk-ant-oat01-test \n"), Model: "claude-sonnet-5-5", Effort: "high"})
	output, err := client.Run(context.Background(), sampleRequest)
	if err != nil || !strings.Contains(string(output), `"summary"`) {
		t.Fatalf("run: %s %v", output, err)
	}
	argv := strings.Split(strings.TrimSpace(read(t, records+"/argv")), "\n")
	equal(t, argv, []string{"-p", "--output-format", "json", "--json-schema", `{"type":"object"}`, "--tools", "",
		"--strict-mcp-config", "--no-session-persistence", "--system-prompt", "You write CVs.",
		"--model", "claude-sonnet-5-5", "--effort", "high"})
	env := read(t, records+"/env")
	if !strings.Contains(env, "CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat01-test\n") || !strings.Contains(env, "DISABLE_AUTOUPDATER=1") ||
		strings.Contains(env, "ANTHROPIC_API_KEY") {
		t.Fatalf("environment: %s", env)
	}
	home := ""
	for _, line := range strings.Split(env, "\n") {
		if strings.HasPrefix(line, "HOME=") {
			home = strings.TrimPrefix(line, "HOME=")
		}
	}
	if home == "" || !strings.HasSuffix(strings.TrimSpace(read(t, records+"/pwd")), "/work") {
		t.Fatalf("an empty HOME and working directory per run: %q %s", home, read(t, records+"/pwd"))
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("the run's directories are removed: %v", err)
	}
	equal(t, read(t, records+"/stdin"), sampleRequest.Prompt)
	equal(t, client.Model(), "claude-sonnet-5-5")
	equal(t, client.HasToken(), true)
}

func TestClaudeReadsTheRealResultOfItsPinnedVersion(t *testing.T) {
	script, _ := fakeClaude(t, fixture(t, "success-2.1.322.json"), 0)
	output, err := claude.New(claude.Config{Binary: script, TokenFile: tokenFile(t, "x")}).Run(context.Background(), sampleRequest)
	if err != nil || !strings.HasPrefix(string(output), `{"greeting":`) {
		t.Fatalf("2.1.322 puts the answer in structured_output: %s %v", output, err)
	}
}

func TestClaudeNamesEachFailure(t *testing.T) {
	token := tokenFile(t, "sk-ant-oat01-test")
	for _, tc := range []struct {
		name     string
		envelope string
		code     int
		want     error
	}{
		{"not logged in (real)", "not-logged-in.json", 1, claude.ErrLoginNeeded},
		{"invalid token (real)", "invalid-token.json", 1, claude.ErrLoginNeeded},
		{"crash without output", "", 2, claude.ErrFailed},
	} {
		envelope := ""
		if tc.envelope != "" {
			envelope = fixture(t, tc.envelope)
		}
		script, _ := fakeClaude(t, envelope, tc.code)
		_, err := claude.New(claude.Config{Binary: script, TokenFile: token}).Run(context.Background(), sampleRequest)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}

	script, _ := fakeClaude(t, fixture(t, "usage-limit.synthetic.json"), 1)
	_, err := claude.New(claude.Config{Binary: script, TokenFile: token}).Run(context.Background(), sampleRequest)
	var limit *claude.LimitError
	if !errors.As(err, &limit) || !limit.Until.After(time.Now()) || limit.Until.After(time.Now().Add(6*time.Hour+time.Minute)) ||
		!strings.Contains(limit.Error(), "usage limit until") {
		t.Fatalf("usage limit: %v", err)
	}
}

func TestClaudeReadsTheResetTime(t *testing.T) {
	now := time.Date(2026, 11, 25, 15, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		text string
		want time.Time
	}{
		{"You've hit your session limit · resets 4:20pm (UTC)", time.Date(2026, 11, 25, 16, 20, 0, 0, time.UTC)},
		{"limit reached, resets 2pm (UTC)", time.Date(2026, 11, 26, 14, 0, 0, 0, time.UTC)},
		{"resets 12am UTC", time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)},
		{"weekly limit · resets Dec 2, 11pm (UTC)", time.Date(2026, 12, 2, 23, 0, 0, 0, time.UTC)},
		{"weekly limit · resets Jan 3rd, 12pm (UTC)", time.Date(2027, 1, 3, 12, 0, 0, 0, time.UTC)},
		{"resets Dec 2, 11pm (UTC) or resets 4pm (UTC)", time.Date(2026, 11, 25, 16, 0, 0, 0, time.UTC)},
		{"usage limit", time.Time{}},
		{"resets 25pm (UTC)", time.Time{}},
		{"resets Foo 2, 1pm (UTC)", time.Time{}},
	} {
		if got := claude.ParseReset(tc.text, now); !got.Equal(tc.want) {
			t.Errorf("%q: %s, want %s", tc.text, got, tc.want)
		}
	}
	equal(t, claude.ClampReset(time.Time{}, now), now.Add(15*time.Minute))
	equal(t, claude.ClampReset(now, now), now.Add(time.Minute))
	equal(t, claude.ClampReset(now.Add(48*time.Hour), now), now.Add(6*time.Hour))
	equal(t, claude.ClampReset(now.Add(time.Hour), now), now.Add(time.Hour))
}

func TestClaudeRefusesAnAnswerWithoutStructuredOutput(t *testing.T) {
	dir := t.TempDir()
	envelope := filepath.Join(dir, "envelope.json")
	if err := os.WriteFile(envelope, []byte(`{"is_error": false, "result": "Here is your CV"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	script, _ := fakeClaude(t, envelope, 0)
	if _, err := claude.New(claude.Config{Binary: script, TokenFile: tokenFile(t, "x")}).Run(context.Background(), sampleRequest); !errors.Is(err, claude.ErrFailed) {
		t.Fatalf("no structured output: %v", err)
	}
	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte(`not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	script, _ = fakeClaude(t, broken, 0)
	if _, err := claude.New(claude.Config{Binary: script, TokenFile: tokenFile(t, "x")}).Run(context.Background(), sampleRequest); !errors.Is(err, claude.ErrFailed) {
		t.Fatalf("unreadable result: %v", err)
	}
}

func TestClaudeNeedsAToken(t *testing.T) {
	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing"), tokenFile(t, "  \n")} {
		client := claude.New(claude.Config{Binary: "/bin/false", TokenFile: path})
		if _, err := client.Run(context.Background(), sampleRequest); !errors.Is(err, claude.ErrNoToken) || client.HasToken() {
			t.Errorf("token file %q: %v", path, err)
		}
	}
}

func TestClaudeStopsAtItsDeadlineAndRefusesHugePrompts(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	client := claude.New(claude.Config{Binary: script, TokenFile: tokenFile(t, "x"), Deadline: 100 * time.Millisecond})
	started := time.Now()
	if _, err := client.Run(context.Background(), sampleRequest); !errors.Is(err, claude.ErrFailed) || time.Since(started) > 10*time.Second {
		t.Fatalf("deadline: %v after %s", err, time.Since(started))
	}
	huge := sampleRequest
	huge.Prompt = strings.Repeat("x", 2<<20)
	if _, err := client.Run(context.Background(), huge); !errors.Is(err, claude.ErrFailed) {
		t.Fatalf("huge prompt: %v", err)
	}
	if _, err := claude.New(claude.Config{Binary: filepath.Join(dir, "missing"), TokenFile: tokenFile(t, "x")}).Run(context.Background(), sampleRequest); !errors.Is(err, claude.ErrFailed) {
		t.Fatalf("missing binary: %v", err)
	}
}

func TestClaudeConfigFromEnv(t *testing.T) {
	for _, name := range []string{"CLAUDE_BIN", "CLAUDE_TOKEN_FILE", "CV_MODEL", "CV_EFFORT", "CLAUDE_DEADLINE"} {
		t.Setenv(name, "")
	}
	equal(t, claude.ConfigFromEnv(), claude.Config{Binary: "claude", Model: claude.DefaultModel, Effort: claude.DefaultEffort, Deadline: claude.DefaultDeadline})
	t.Setenv("CLAUDE_BIN", "/usr/local/bin/claude")
	t.Setenv("CLAUDE_TOKEN_FILE", "/run/secrets/hoply/claude_oauth_token")
	t.Setenv("CV_MODEL", "claude-opus-5-5")
	t.Setenv("CV_EFFORT", "medium")
	t.Setenv("CLAUDE_DEADLINE", "90s")
	equal(t, claude.ConfigFromEnv(), claude.Config{Binary: "/usr/local/bin/claude", TokenFile: "/run/secrets/hoply/claude_oauth_token",
		Model: "claude-opus-5-5", Effort: "medium", Deadline: 90 * time.Second})
	t.Setenv("CLAUDE_DEADLINE", "soon")
	equal(t, claude.ConfigFromEnv().Deadline, claude.DefaultDeadline)
	equal(t, len(claude.New(claude.Config{}).Model()), 0)
}
