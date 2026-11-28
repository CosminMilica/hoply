package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"syscall"
	"testing"
	"time"

	"jobs/tests/testutil"
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

func reasonOf(body map[string]any) any {
	envelope, _ := body["error"].(map[string]any)
	details, _ := envelope["context"].(map[string]any)
	return details["reason"]
}

const romanianAd = "DevOps Engineer\nFirma: Carpathia Systems, Brașov (hibrid)\n" +
	"Căutăm un coleg care să ne ajute să automatizăm infrastructura și să avem lansări mai sigure.\n" +
	"Responsabilități\n- Administrezi servere Linux și containere Docker.\n- Automatizezi infrastructura cu Ansible și Terraform.\n" +
	"Cerințe\n- Minimum 3 ani experiență în DevOps.\n- Bash sau Python pentru automatizări.\nConstituie un avantaj\n- Kubernetes\n- AWS"

func TestJobsContractEndToEnd(t *testing.T) {
	schema := testutil.NewSchema(t)
	migrateBinaryOK(t, schema.URL)
	jobs := startServer(t, schema) + "/jobs"

	status, list := call(t, http.MethodGet, jobs, nil)
	if status != http.StatusOK || list["total"] != float64(0) || fmt.Sprint(list["items"]) != "[]" {
		t.Fatalf("GET /jobs starts empty: %d %v", status, list)
	}
	var catalogID string
	if err := schema.Admin.QueryRow(context.Background(), "INSERT INTO "+schema.Name+".jobs "+
		"(title, company, city, country, work_mode, level, source, created_at) "+
		"VALUES ('Data Engineer', 'Northwind Data', 'Iași', 'Romania', 'onsite', 'senior', 'catalog', NOW() - INTERVAL '1 hour') "+
		"RETURNING id::text").Scan(&catalogID); err != nil {
		t.Fatal(err)
	}
	if status, list := call(t, http.MethodGet, jobs, nil); status != http.StatusOK || list["total"] != float64(1) {
		t.Fatalf("GET /jobs with a catalog job: %d %v", status, list)
	}
	if status, list := call(t, http.MethodGet, jobs+"?city=Iasi&level=senior", nil); status != http.StatusOK || list["total"] != float64(1) {
		t.Fatalf("GET /jobs filtered: %d %v", status, list)
	}
	if status, body := call(t, http.MethodGet, jobs+"?level=boss", nil); status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "VALIDATION_FAILED" {
		t.Fatalf("bad level: %d %v", status, body)
	}

	status, draft := call(t, http.MethodPost, jobs+"/read", map[string]any{"text": romanianAd})
	if status != http.StatusOK || draft["company"] != "Carpathia Systems" || draft["city"] != "Brașov" || draft["work_mode"] != "hybrid" ||
		fmt.Sprint(draft["nice_to_have"]) != "[Kubernetes AWS]" || draft["source"] != "text" || draft["host"] != "" {
		t.Fatalf("POST /jobs/read text: %d %v", status, draft)
	}
	for link, reason := range map[string]string{
		"https://www.linkedin.com/jobs/view/123": "login_required",
		"http://127.0.0.1:1/job":                 "blocked_address",
		"http://localhost/job":                   "blocked_address",
		"http://[::1]/job":                       "blocked_address",
		"http://169.254.169.254/latest":          "blocked_address",
	} {
		status, body := call(t, http.MethodPost, jobs+"/read", map[string]any{"url": link})
		if status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "PAGE_UNREADABLE" || reasonOf(body) != reason {
			t.Fatalf("POST /jobs/read %s: %d %v", link, status, body)
		}
	}

	delete(draft, "host")
	status, created := call(t, http.MethodPost, jobs, draft)
	if status != http.StatusCreated || created["added_by_me"] != true || created["read_from"] != "text" {
		t.Fatalf("POST /jobs: %d %v", status, created)
	}
	if status, body := call(t, http.MethodPost, jobs, map[string]any{"title": "x", "source": "catalog", "read_from": "text"}); status != http.StatusUnprocessableEntity {
		t.Fatalf("POST a catalog job: %d %v", status, body)
	}
	_, list = call(t, http.MethodGet, jobs, nil)
	if list["total"] != float64(2) || list["items"].([]any)[0].(map[string]any)["id"] != created["id"] {
		t.Fatalf("my job comes first: %v", list)
	}
	path := jobs + "/" + created["id"].(string)
	if status, got := call(t, http.MethodGet, path, nil); status != http.StatusOK || got["title"] != "DevOps Engineer" {
		t.Fatalf("GET /jobs/{id}: %d %v", status, got)
	}
	if status, _ := call(t, http.MethodDelete, path, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: %d", status)
	}
	for _, target := range []string{path, jobs + "/not-a-uuid", jobs + "/" + catalogID} {
		if status, body := call(t, http.MethodDelete, target, nil); status != http.StatusNotFound || testutil.ErrorCode(body) != "NOT_FOUND" {
			t.Fatalf("DELETE %s: %d %v", target, status, body)
		}
	}
	if status, _ := call(t, http.MethodGet, path, nil); status != http.StatusNotFound {
		t.Fatalf("GET deleted: %d", status)
	}
}
