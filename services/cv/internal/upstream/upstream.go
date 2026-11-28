// Package upstream reads the profile and the jobs a CV is made from, from the
// profile and jobs services.
package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	platformauth "forge.local/platform-auth-go"
	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/identity"
	"cv/internal/models"
)

const maxBody = 4 << 20

// Client calls the profile and jobs services' APIs (docs/api/profile.md,
// docs/api/jobs.md).
type Client struct {
	profileURL string
	jobsURL    string
	http       *http.Client
	// profileTokens and jobsTokens mint the service tokens each service takes, for the tenant in the context; nil
	// without the gatekeeper's credentials, when requests go without a token.
	profileTokens Minter
	jobsTokens    Minter
}

// Minter mints a service token; *platformauth.S2SClient is one.
type Minter interface {
	Token(ctx context.Context, options platformauth.S2SRequestOptions) (string, error)
}

// WithTokens calls the profile and jobs services with the tokens of the given minters.
func (c *Client) WithTokens(profile, jobs Minter) *Client {
	c.profileTokens, c.jobsTokens = profile, jobs
	return c
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
// services' local ports. With GATEKEEPER_CLIENT_ID, GATEKEEPER_CLIENT_SECRET and
// GATEKEEPER_TOKEN_ENDPOINT it calls them with the gatekeeper's service tokens.
func FromEnv() (*Client, error) {
	client := New(envOr("PROFILE_SERVICE_URL", "http://localhost:5010"), envOr("JOBS_SERVICE_URL", "http://localhost:5020"))
	id, secret, endpoint := os.Getenv("GATEKEEPER_CLIENT_ID"), os.Getenv("GATEKEEPER_CLIENT_SECRET"), os.Getenv("GATEKEEPER_TOKEN_ENDPOINT")
	if id == "" && secret == "" && endpoint == "" {
		return client, nil
	}
	minter := func(audience string) (Minter, error) {
		return platformauth.NewS2SClient(platformauth.S2SClientConfig{Audience: audience, TokenEndpoint: endpoint, ClientID: id, ClientSecret: secret})
	}
	profile, err := minter("svc-profile")
	if err != nil {
		return nil, err
	}
	jobs, err := minter("svc-jobs")
	if err != nil {
		return nil, err
	}
	return client.WithTokens(profile, jobs), nil
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
	raw, status, err := c.get(ctx, c.profileTokens, c.profileURL+"/api/v1/profiles/me")
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
	raw, status, err := c.get(ctx, c.jobsTokens, c.jobsURL+"/api/v1/jobs/"+id.String())
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

func (c *Client) get(ctx context.Context, tokens Minter, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if err := authorize(ctx, req, tokens); err != nil {
		return nil, 0, err
	}
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
	if err := authorize(ctx, req, c.jobsTokens); err != nil {
		return err
	}
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

// authorize adds a service token for the tenant of the context's identity: the caller's, or the one the worker writes
// for.
func authorize(ctx context.Context, req *http.Request, tokens Minter) error {
	if tokens == nil {
		return nil
	}
	ident, ok := identity.FromContext(ctx)
	if !ok {
		return errors.New("no tenant to call the service for")
	}
	token, err := tokens.Token(ctx, platformauth.S2SRequestOptions{TenantID: ident.TenantID.String()})
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}
