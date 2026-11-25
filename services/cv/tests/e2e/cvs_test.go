package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"cv/tests/testutil"
)

const jobID = "00000000-0000-4000-8000-000000000001"

// sourcesServer plays the profile and jobs services with the test data.
func sourcesServer(t *testing.T) string {
	t.Helper()
	profile, err := os.ReadFile("../testdata/profile.json")
	if err != nil {
		t.Fatal(err)
	}
	var jobs []json.RawMessage
	raw, err := os.ReadFile("../testdata/jobs.json")
	if err != nil || json.Unmarshal(raw, &jobs) != nil {
		t.Fatalf("jobs test data: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/profiles/me":
			_, _ = w.Write(profile)
		case "/api/v1/jobs/" + jobID:
			_, _ = w.Write(jobs[0])
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// startServer runs the service binary against databaseURL, reading the profile
// and the jobs from sources, with extra environment, and returns the /api/v1 base url.
func startServer(t *testing.T, databaseURL, sources string, extra ...string) string {
	t.Helper()
	port := freePort(t)
	server := command(t, "server", append(append(serverEnv(t), "ENV=testing", "PORT="+strconv.Itoa(port),
		"DATABASE_URL="+databaseURL, "PROFILE_SERVICE_URL="+sources, "JOBS_SERVICE_URL="+sources), extra...)...)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = server.Process.Signal(syscall.SIGTERM)
		if err := server.Wait(); err != nil {
			t.Errorf("graceful shutdown must exit 0: %v", err)
		}
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port) + "/api/v1"
	for range 200 {
		if response, err := http.Get(base + "/health/live"); err == nil {
			response.Body.Close()
			return base
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("service did not start")
	return ""
}

func call(t *testing.T, method, url string, payload any) (int, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatal(err)
		}
	}
	request, err := http.NewRequest(method, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	decoded := map[string]any{}
	if response.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
			t.Fatalf("%s %s: %d non-JSON response: %v", method, url, response.StatusCode, err)
		}
	}
	return response.StatusCode, decoded
}

// waitReady asks for the CV until it is ready or failed.
func waitReady(t *testing.T, url string) map[string]any {
	t.Helper()
	var cv map[string]any
	for range 200 {
		_, cv = call(t, http.MethodGet, url, nil)
		if cv["status"] == "ready" || cv["status"] == "failed" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	return cv
}

func TestCvWrittenByClaudeEndToEnd(t *testing.T) {
	dir := t.TempDir()
	envelope, err := filepath.Abs("../testdata/claude/success.synthetic.json")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat > /dev/null\ncat '"+envelope+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	token := filepath.Join(dir, "claude_oauth_token")
	if err := os.WriteFile(token, []byte("sk-ant-oat01-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	schema := testutil.NewSchema(t)
	migrateBinaryOK(t, schema.URL)
	base := startServer(t, schema.URL, sourcesServer(t), "CV_WRITER=claude-cli", "CLAUDE_BIN="+script, "CLAUDE_TOKEN_FILE="+token)

	if _, writer := call(t, http.MethodGet, base+"/writer", nil); writer["writer"] != "claude" || writer["state"] != "ok" {
		t.Fatalf("GET /writer: %v", writer)
	}
	_, queued := call(t, http.MethodPost, base+"/cvs", map[string]any{"job_id": jobID})
	cv := waitReady(t, base+"/cvs/"+queued["id"].(string))
	tailored, _ := cv["tailored"].(map[string]any)
	if cv["written_by"] != "claude" || cv["model"] != "claude-sonnet-5-5" || tailored["summary"] == nil {
		t.Fatalf("written by Claude: %v", cv)
	}
}

func TestCvContractEndToEnd(t *testing.T) {
	schema := testutil.NewSchema(t)
	migrateBinaryOK(t, schema.URL)
	base := startServer(t, schema.URL, sourcesServer(t))

	status, questions := call(t, http.MethodGet, base+"/jobs/"+jobID+"/questions", nil)
	if status != http.StatusOK || questions["base_score"] != float64(56) {
		t.Fatalf("questions: %d %v", status, questions)
	}
	status, queued := call(t, http.MethodPost, base+"/cvs", map[string]any{"job_id": jobID, "skip_roles": []string{}, "focus": []string{}, "confirmed": []int{1}})
	if status != http.StatusAccepted || queued["status"] != "queued" || queued["file_name"] != "Andrei_Popescu_CV_Capgemini_Romania.pdf" {
		t.Fatalf("POST /cvs: %d %v", status, queued)
	}
	cv := base + "/cvs/" + queued["id"].(string)
	ready := waitReady(t, cv)
	tailored, _ := ready["tailored"].(map[string]any)
	if ready["status"] != "ready" || tailored["headline"] != "DevOps Engineer" || ready["written_by"] != "rules" ||
		ready["rules_reason"] != "off" {
		t.Fatalf("the worker in the binary writes it: %v", ready)
	}
	if _, list := call(t, http.MethodGet, base+"/cvs?job_id="+jobID, nil); len(list["items"].([]any)) != 1 {
		t.Fatalf("GET /cvs: %v", list)
	}
	if status, body := call(t, http.MethodPost, base+"/cvs", map[string]any{"job_id": "00000000-0000-4000-8000-000000000009"}); status != http.StatusNotFound ||
		testutil.ErrorCode(body) != "NOT_FOUND" {
		t.Fatalf("unknown job: %d %v", status, body)
	}
	if status, _ := call(t, http.MethodDelete, cv, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: %d", status)
	}
	if status, _ := call(t, http.MethodGet, cv, nil); status != http.StatusNotFound {
		t.Fatalf("GET deleted: %d", status)
	}
}
