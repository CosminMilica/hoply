package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	smartRecruitersPage = 100
	// maxSmartRecruitersJobs caps one board's list.
	maxSmartRecruitersJobs = 500
)

// smartRecruitersList lists a SmartRecruiters company's jobs in the country,
// without their descriptions. complete is false when the list stopped at
// maxSmartRecruitersJobs.
func (r *Reader) smartRecruitersList(ctx context.Context, company string, country Country) (postings []Posting, complete bool, err error) {
	postings = []Posting{}
	for offset := 0; ; offset += smartRecruitersPage {
		if offset >= maxSmartRecruitersJobs {
			return postings, false, nil
		}
		api := fmt.Sprintf("https://api.smartrecruiters.com/v1/companies/%s/postings?country=%s&limit=%d&offset=%d",
			url.PathEscape(company), url.QueryEscape(strings.ToLower(country.Code)), smartRecruitersPage, offset)
		body, err := r.json(ctx, http.MethodGet, api, nil)
		if err != nil {
			return nil, false, err
		}
		var list struct {
			TotalFound int `json:"totalFound"`
			Content    []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				ReleasedDate string `json:"releasedDate"`
				Company      struct {
					Name string `json:"name"`
				} `json:"company"`
				Location struct {
					City         string `json:"city"`
					Country      string `json:"country"`
					FullLocation string `json:"fullLocation"`
					Remote       bool   `json:"remote"`
					Hybrid       bool   `json:"hybrid"`
				} `json:"location"`
			} `json:"content"`
		}
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, false, fail(NoJobData, err)
		}
		for _, job := range list.Content {
			id, where := job.ID, job.Location
			mode := ""
			switch {
			case where.Remote:
				mode = "remote"
			case where.Hybrid:
				mode = "hybrid"
			}
			postings = append(postings, Posting{ID: id, URL: "https://jobs.smartrecruiters.com/" + url.PathEscape(company) + "/" + url.PathEscape(id),
				PostedAt: timeOf(job.ReleasedDate), Places: []string{text(where.FullLocation), text(where.City)}, InCountry: true,
				Page: Page{Title: text(job.Name), Company: text(job.Company.Name), City: text(where.City), Country: text(where.Country), WorkMode: mode},
				describe: func(ctx context.Context, r *Reader) (*Page, []string, *time.Time, error) {
					page, err := r.readBoard(ctx, smartRecruiters(company, id))
					return page, nil, nil, err
				}})
		}
		if len(list.Content) < smartRecruitersPage || offset+smartRecruitersPage >= list.TotalFound {
			return postings, true, nil
		}
	}
}
