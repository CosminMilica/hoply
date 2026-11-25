// Package upstream reads the profile and the jobs a CV is made from, from the
// profile and jobs services.
package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/models"
)

const maxBody = 4 << 20

// Client calls the profile and jobs services' APIs (docs/api/profile.md,
// docs/api/jobs.md).
type Client struct {
	profileURL string
	jobsURL    string
	http       *http.Client
}

// New takes the services' base URLs, such as http://profile:5010.
func New(profileURL, jobsURL string) *Client {
	return &Client{
		profileURL: strings.TrimRight(profileURL, "/"),
		jobsURL:    strings.TrimRight(jobsURL, "/"),
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

// FromEnv reads PROFILE_SERVICE_URL and JOBS_SERVICE_URL, defaulting to the
// services' local ports.
func FromEnv() *Client {
	return New(envOr("PROFILE_SERVICE_URL", "http://localhost:5010"), envOr("JOBS_SERVICE_URL", "http://localhost:5020"))
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

// Profile returns the caller's profile, and the profile as the service sent it.
func (c *Client) Profile(ctx context.Context) (models.Profile, json.RawMessage, error) {
	var profile models.Profile
	raw, status, err := c.get(ctx, c.profileURL+"/api/v1/profiles/me")
	if err != nil || status != http.StatusOK {
		return profile, nil, unavailable("profile", status, err)
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		return profile, nil, unavailable("profile", status, err)
	}
	return profile, raw, nil
}

// Job returns the job, and the job as the service sent it; a job the service
// does not know is a not-found error.
func (c *Client) Job(ctx context.Context, id uuid.UUID) (models.Job, json.RawMessage, error) {
	var job models.Job
	raw, status, err := c.get(ctx, c.jobsURL+"/api/v1/jobs/"+id.String())
	if err == nil && status == http.StatusNotFound {
		return job, nil, apperrors.NewNotFound("Job", id.String())
	}
	if err != nil || status != http.StatusOK {
		return job, nil, unavailable("jobs", status, err)
	}
	if err := json.Unmarshal(raw, &job); err != nil {
		return job, nil, unavailable("jobs", status, err)
	}
	return job, raw, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	return body, res.StatusCode, err
}

func unavailable(dependency string, status int, cause error) error {
	if cause == nil {
		cause = fmt.Errorf("answered %d", status)
	}
	slog.Warn("dependency failed", "dependency", dependency, "error", cause)
	return apperrors.NewDependencyUnavailable(dependency)
}

// SetRequirements replaces the job's requirements with the analysis's reading of its ad.
func (c *Client) SetRequirements(ctx context.Context, id uuid.UUID, must, nice []string) error {
	body, err := json.Marshal(map[string][]string{"must_have": must, "nice_to_have": nice})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.jobsURL+"/api/v1/jobs/"+id.String()+"/requirements", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("jobs answered %d", res.StatusCode)
	}
	return nil
}
