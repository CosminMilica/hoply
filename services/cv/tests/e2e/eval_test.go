package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// evalRun runs the eval binary without Claude: the writer falls back to the rules and the judge answers nothing.
func evalRun(t *testing.T, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := command(t, "eval", append([]string{"CLAUDE_TOKEN_FILE=", "CLAUDE_BIN=/nonexistent/claude"}, env...)...)
	cmd.Args = append(cmd.Args, args...)
	out, err := cmd.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); ok {
		return string(out), exit.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

func TestEvalBinaryWritesJudgesAndComparesRuns(t *testing.T) {
	run := filepath.Join(t.TempDir(), "run")
	out, code := evalRun(t, nil, "write", "-cases", "eval/cases", "-out", run, "-only", "finance-fpa-analyst")
	if code != 0 || !strings.Contains(out, "| finance-fpa-analyst | rules |") {
		t.Fatalf("write: %d %s", code, out)
	}
	for _, args := range [][]string{{"show", "-run", run}, {"judge", "-cases", "eval/cases", "-run", run}} {
		if out, code := evalRun(t, nil, args...); code != 0 || !strings.Contains(out, "finance-fpa-analyst") {
			t.Fatalf("%s: %d %s", args[0], code, out)
		}
	}
	if out, code := evalRun(t, nil, "compare", "-cases", "eval/cases", "-a", run, "-b", run); code != 0 || !strings.Contains(out, "| **total** | a 0 · b 0 · tie 0 |") {
		t.Fatalf("compare: %d %s", code, out)
	}
	for _, args := range [][]string{{}, {"nope"}, {"write", "-cases", "eval/cases", "-only", "nobody", "-out", run}, {"show", "-run", t.TempDir()},
		{"judge", "-run", t.TempDir()}, {"compare", "-a", t.TempDir(), "-b", run}, {"compare", "-a", run, "-b", t.TempDir()}} {
		if _, code := evalRun(t, nil, args...); code != 1 {
			t.Fatalf("%v exits 1, got %d", args, code)
		}
	}
}

func TestEvalBinaryExportsCasesFromTheServices(t *testing.T) {
	id := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/profiles/me":
			_, _ = w.Write([]byte(`{"name": "Ana Pop", "skills": ["Go"]}`))
		case "/api/v1/jobs/" + id:
			_, _ = w.Write([]byte(`{"id": "` + id + `", "title": "Platform Engineer", "company": "Acme Cloud"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := filepath.Join(t.TempDir(), "local")
	env := []string{"PROFILE_SERVICE_URL=" + server.URL, "JOBS_SERVICE_URL=" + server.URL}
	if out, code := evalRun(t, env, "export", "-ids", id, "-out", dir); code != 0 || !strings.Contains(out, "saved acme-cloud-platform-engineer") {
		t.Fatalf("export: %d %s", code, out)
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "acme-cloud-platform-engineer.json")); err != nil || !strings.Contains(string(raw), "Ana Pop") {
		t.Fatalf("case: %s %v", raw, err)
	}
	if _, code := evalRun(t, env, "export", "-ids", uuid.NewString(), "-out", dir); code != 1 {
		t.Fatalf("an unknown job exits 1, got %d", code)
	}
	if _, code := evalRun(t, []string{"PROFILE_SERVICE_URL=" + server.URL + "/nowhere"}, "export", "-ids", id); code != 1 {
		t.Fatalf("a missing profile exits 1, got %d", code)
	}
}
