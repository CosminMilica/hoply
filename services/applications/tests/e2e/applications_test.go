package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"

	"applications/tests/testutil"
)

// startServer runs the service binary against databaseURL and returns the
// /api/v1 base url; the server is stopped when the test ends.
func startServer(t *testing.T, schema testutil.TestSchema) string {
	t.Helper()
	port := freePort(t)
	server := command(t, "server", append(append(serverEnv(t), asPerson(t, schema)...), "ENV=testing", "PORT="+strconv.Itoa(port))...)
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
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
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

func TestApplicationsContractEndToEnd(t *testing.T) {
	schema := testutil.NewSchema(t)
	migrateBinaryOK(t, schema.URL)
	base := startServer(t, schema)
	applications, skips := base+"/applications", base+"/skips"

	if status, list := call(t, http.MethodGet, applications, nil); status != http.StatusOK || fmt.Sprint(list["items"]) != "[]" {
		t.Fatalf("GET /applications: %d %v", status, list)
	}
	first, second := uuid.NewString(), uuid.NewString()
	status, saved := call(t, http.MethodPut, applications+"/"+first, map[string]any{"status": "saved"})
	if status != http.StatusOK || saved["status"] != "saved" || saved["saved_at"] == nil || saved["cv_ready_at"] != nil {
		t.Fatalf("PUT saved: %d %v", status, saved)
	}
	call(t, http.MethodPut, applications+"/"+second, map[string]any{"status": "cv_ready"})
	call(t, http.MethodPut, applications+"/"+first, map[string]any{"status": "applied"})
	status, kept := call(t, http.MethodPut, applications+"/"+first, map[string]any{"status": "cv_ready"})
	if status != http.StatusOK || kept["status"] != "applied" || kept["applied_at"] == nil || kept["cv_ready_at"] == nil ||
		kept["saved_at"] != saved["saved_at"] {
		t.Fatalf("a new CV on an applied job: %d %v", status, kept)
	}
	_, list := call(t, http.MethodGet, applications, nil)
	items := list["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["job_id"] != first {
		t.Fatalf("most recently changed first: %v", list)
	}
	for _, tc := range []struct {
		path    string
		payload any
		field   string
	}{
		{applications + "/" + first, map[string]any{"status": "hired"}, "status"},
		{applications + "/not-a-uuid", map[string]any{"status": "saved"}, "job_id"},
	} {
		status, body := call(t, http.MethodPut, tc.path, tc.payload)
		envelope, _ := body["error"].(map[string]any)
		details, _ := envelope["context"].(map[string]any)
		if status != http.StatusUnprocessableEntity || details["field"] != tc.field {
			t.Fatalf("PUT %s: %d %v", tc.path, status, body)
		}
	}
	if status, _ := call(t, http.MethodDelete, applications+"/"+second, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: %d", status)
	}
	if status, body := call(t, http.MethodDelete, applications+"/"+second, nil); status != http.StatusNotFound || testutil.ErrorCode(body) != "NOT_FOUND" {
		t.Fatalf("DELETE again: %d %v", status, body)
	}

	for _, job := range []string{first, second, first} {
		if status, _ := call(t, http.MethodPut, skips+"/"+job, nil); status != http.StatusNoContent {
			t.Fatalf("PUT /skips: %d", status)
		}
	}
	if _, list := call(t, http.MethodGet, skips, nil); fmt.Sprint(list["job_ids"]) != fmt.Sprint([]string{first, second}) {
		t.Fatalf("GET /skips: %v", list)
	}
	if status, _ := call(t, http.MethodDelete, skips+"/"+uuid.NewString(), nil); status != http.StatusNoContent {
		t.Fatalf("DELETE a skip that is not there: %d", status)
	}
	if status, _ := call(t, http.MethodDelete, skips, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE /skips: %d", status)
	}
	if _, list := call(t, http.MethodGet, skips, nil); fmt.Sprint(list["job_ids"]) != "[]" {
		t.Fatalf("after clear: %v", list)
	}
}
