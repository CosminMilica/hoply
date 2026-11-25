package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"profile/tests/testutil"
)

// startServer runs the service binary against databaseURL and returns the
// /api/v1 base url; the server is stopped when the test ends.
func startServer(t *testing.T, databaseURL string) string {
	t.Helper()
	port := freePort(t)
	server := command(t, "server", append(serverEnv(t), "ENV=testing", "PORT="+strconv.Itoa(port),
		"DATABASE_URL="+databaseURL)...)
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

func TestProfileContractEndToEnd(t *testing.T) {
	schema := testutil.NewSchema(t)
	migrateBinaryOK(t, schema.URL)
	me := startServer(t, schema.URL) + "/profiles/me"

	status, profile := call(t, http.MethodGet, me, nil)
	if status != http.StatusOK || profile["name"] != "" || fmt.Sprint(profile["experience"]) != "[]" ||
		fmt.Sprint(profile["projects"]) != "[]" || profile["salary_min"] != nil || profile["open_to_relocation"] != false ||
		profile["created_at"] == nil || profile["updated_at"] == nil {
		t.Fatalf("GET /me creates the empty profile: %d %v", status, profile)
	}

	status, patched := call(t, http.MethodPatch, me, map[string]any{
		"name": " Ana Pop ", "headline": "Backend engineer", "city": "Cluj", "work_modes": []string{"remote", "hybrid", "remote"},
		"notice_period_weeks": 4, "salary_min": 5000, "salary_currency": "eur", "salary_period": "month",
		"employment_types": []string{"full_time"}, "available_from": "2027-02", "additional_info": []string{" Has a car ", ""},
	})
	if status != http.StatusOK || patched["name"] != "Ana Pop" || fmt.Sprint(patched["work_modes"]) != "[remote hybrid]" ||
		patched["salary_currency"] != "EUR" || patched["notice_period_weeks"] != float64(4) ||
		fmt.Sprint(patched["additional_info"]) != "[Has a car]" || patched["id"] != profile["id"] {
		t.Fatalf("PATCH /me: %d %v", status, patched)
	}
	status, cleared := call(t, http.MethodPatch, me, map[string]any{"notice_period_weeks": nil})
	if status != http.StatusOK || cleared["notice_period_weeks"] != nil || cleared["salary_min"] != float64(5000) || cleared["city"] != "Cluj" {
		t.Fatalf("PATCH null clears only that field: %d %v", status, cleared)
	}
	for _, body := range []map[string]any{{"email": "not-an-email"}, {"name": strings.Repeat("n", 121)},
		{"notice_period_weeks": 53}, {"work_modes": []string{"office"}}, {"salary_currency": "EURO"}} {
		if status, body := call(t, http.MethodPatch, me, body); status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "VALIDATION_FAILED" {
			t.Fatalf("invalid patch: %d %v", status, body)
		}
	}

	status, skilled := call(t, http.MethodPut, me+"/skills", map[string]any{"skills": []string{"Go", " ", "TypeScript", "go"}})
	if status != http.StatusOK || fmt.Sprint(skilled["skills"]) != "[Go TypeScript]" {
		t.Fatalf("PUT /me/skills: %d %v", status, skilled)
	}

	status, older := call(t, http.MethodPost, me+"/experience", map[string]any{
		"role": "Developer", "company": "Acme", "location": "Cluj", "employment_type": "full_time", "industry": "Fintech",
		"team_size": 6, "start": "2016-03", "end": "2019-08", "tools": []string{"PHP", "php", "MySQL"},
		"bullets": []map[string]any{{"text": "Built billing", "result": "-20% churn", "skills": []string{"PHP"}}, {"text": " "}},
	})
	if status != http.StatusCreated || fmt.Sprint(older["tools"]) != "[PHP MySQL]" || older["team_size"] != float64(6) ||
		fmt.Sprint(older["bullets"]) != "[map[result:-20% churn skills:[PHP] text:Built billing]]" {
		t.Fatalf("POST /me/experience: %d %v", status, older)
	}
	_, newer := call(t, http.MethodPost, me+"/experience", map[string]any{"role": "Lead", "start": "2019-09", "end": ""})
	olderPath := me + "/experience/" + older["id"].(string)
	status, updated := call(t, http.MethodPut, olderPath, map[string]any{"role": "Senior developer", "start": "2016-03", "end": "2019-08"})
	if status != http.StatusOK || updated["role"] != "Senior developer" || updated["id"] != older["id"] || updated["team_size"] != nil {
		t.Fatalf("PUT /me/experience/{id}: %d %v", status, updated)
	}
	if status, body := call(t, http.MethodPut, olderPath, map[string]any{"role": "x", "start": "2020-02", "end": "2020-01"}); status != http.StatusUnprocessableEntity ||
		body["error"].(map[string]any)["context"].(map[string]any)["field"] != "end" {
		t.Fatalf("end before start: %d %v", status, body)
	}

	_, alpha := call(t, http.MethodPost, me+"/projects", map[string]any{"name": "Alpha", "url": "https://alpha.dev"})
	status, beta := call(t, http.MethodPost, me+"/projects", map[string]any{"name": "Beta", "bullets": []map[string]any{{"text": "Launched"}}})
	if status != http.StatusCreated || fmt.Sprint(beta["bullets"]) != "[map[result: skills:[] text:Launched]]" {
		t.Fatalf("POST /me/projects: %d %v", status, beta)
	}
	alphaPath := me + "/projects/" + alpha["id"].(string)
	if status, edited := call(t, http.MethodPut, alphaPath, map[string]any{"name": "Alpha 2"}); status != http.StatusOK || edited["url"] != "" {
		t.Fatalf("PUT /me/projects/{id}: %d %v", status, edited)
	}

	_, bsc := call(t, http.MethodPost, me+"/education", map[string]any{"level": "bachelor", "degree": "BSc", "school": "UBB", "start_year": "2012", "end_year": "2015"})
	if status, msc := call(t, http.MethodPost, me+"/education", map[string]any{"level": "master"}); status != http.StatusCreated || msc["degree"] != "" {
		t.Fatalf("POST /me/education: %d %v", status, msc)
	}
	bscPath := me + "/education/" + bsc["id"].(string)
	if status, edited := call(t, http.MethodPut, bscPath, map[string]any{"level": "bachelor", "field": "Computer Science", "start_year": "2012"}); status != http.StatusOK ||
		edited["field"] != "Computer Science" || edited["end_year"] != "" {
		t.Fatalf("PUT /me/education/{id}: %d %v", status, edited)
	}
	if status, _ := call(t, http.MethodPost, me+"/education", map[string]any{"degree": "BSc"}); status != http.StatusUnprocessableEntity {
		t.Fatalf("education without level: %d", status)
	}

	_, cka := call(t, http.MethodPost, me+"/certifications", map[string]any{"name": "CKA", "issuer": "CNCF", "issued": "2025-01"})
	_, ckad := call(t, http.MethodPost, me+"/certifications", map[string]any{"name": "CKAD"})
	ckaPath := me + "/certifications/" + cka["id"].(string)
	if status, edited := call(t, http.MethodPut, ckaPath, map[string]any{"name": "CKA", "issued": "2025-01", "expires": "2028-01"}); status != http.StatusOK ||
		edited["expires"] != "2028-01" || edited["issuer"] != "" {
		t.Fatalf("PUT /me/certifications/{id}: %d %v", status, edited)
	}

	_, romanian := call(t, http.MethodPost, me+"/languages", map[string]any{"name": "Romanian", "level": "Native"})
	if status, _ := call(t, http.MethodPost, me+"/languages", map[string]any{"name": "German", "level": "B3"}); status != http.StatusUnprocessableEntity {
		t.Fatalf("unknown level: %d", status)
	}

	_, profile = call(t, http.MethodGet, me, nil)
	first := func(key string) any { return profile[key].([]any)[0].(map[string]any)["id"] }
	if first("experience") != newer["id"] || first("projects") != beta["id"] || first("education") != bsc["id"] ||
		first("certifications") != cka["id"] || first("languages") != romanian["id"] ||
		profile["certifications"].([]any)[1].(map[string]any)["id"] != ckad["id"] {
		t.Fatalf("order: %v", profile)
	}

	for _, path := range []string{olderPath, alphaPath, bscPath, ckaPath, me + "/languages/" + romanian["id"].(string)} {
		if status, _ := call(t, http.MethodDelete, path, nil); status != http.StatusNoContent {
			t.Fatalf("DELETE %s: %d", path, status)
		}
		if status, body := call(t, http.MethodDelete, path, nil); status != http.StatusNotFound || testutil.ErrorCode(body) != "NOT_FOUND" {
			t.Fatalf("DELETE %s again: %d %v", path, status, body)
		}
	}
	for _, path := range []string{olderPath, alphaPath, bscPath, ckaPath, me + "/projects/not-a-uuid"} {
		if status, _ := call(t, http.MethodPut, path, map[string]any{"name": "x", "role": "x", "level": "other"}); status != http.StatusNotFound {
			t.Fatalf("PUT %s after delete: %d", path, status)
		}
	}
	_, profile = call(t, http.MethodGet, me, nil)
	for _, key := range []string{"experience", "projects", "education", "certifications"} {
		if len(profile[key].([]any)) != 1 {
			t.Fatalf("after deletes %s: %v", key, profile[key])
		}
	}
}
