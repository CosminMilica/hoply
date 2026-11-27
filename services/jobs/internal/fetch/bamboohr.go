package fetch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BambooHR serves each company's careers page from <company>.bamboohr.com: /careers/list lists the openings with
// their city, /careers/<id>/detail has one opening's description, country and posting date.
const bambooHRHost = ".bamboohr.com"

func bambooHRAPI(company string) string {
	return "https://" + url.PathEscape(company) + bambooHRHost + "/careers"
}

// bambooHRCompany is the company a BambooHR host belongs to, its subdomain, or "".
func bambooHRCompany(host string) string {
	company, ok := strings.CutSuffix(host, bambooHRHost)
	if !ok || strings.Contains(company, ".") {
		return ""
	}
	return company
}

func bambooHRBoardOf(host string, _ []string, _ url.Values) (string, bool) {
	company := bambooHRCompany(host)
	return company, company != ""
}

type bambooHRLocation struct {
	City           string `json:"city"`
	State          string `json:"state"`
	AddressCountry string `json:"addressCountry"`
}

// bambooHRWorkMode maps BambooHR's location type: 0 on site, 1 remote, 2 hybrid.
func bambooHRWorkMode(locationType string, remote *bool) string {
	switch {
	case remote != nil && *remote, locationType == "1":
		return "remote"
	case locationType == "2":
		return "hybrid"
	case locationType == "0":
		return "onsite"
	}
	return ""
}

// bambooHRList lists a company's openings without their descriptions.
func (r *Reader) bambooHRList(ctx context.Context, company string, _ Country) ([]Posting, bool, error) {
	return r.apiList(ctx, bambooHRAPI(company)+"/list", func(body []byte) ([]Posting, error) {
		var list struct {
			Result []struct {
				ID           string           `json:"id"`
				Name         string           `json:"jobOpeningName"`
				Location     bambooHRLocation `json:"location"`
				IsRemote     *bool            `json:"isRemote"`
				LocationType string           `json:"locationType"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, err
		}
		postings := []Posting{}
		for _, job := range list.Result {
			id := job.ID
			postings = append(postings, Posting{ID: id, URL: bambooHRAPI(company) + "/" + url.PathEscape(id),
				Places: []string{text(job.Location.City), text(job.Location.State)},
				Page: Page{Title: text(job.Name), Company: slugName(company), City: text(job.Location.City),
					WorkMode: bambooHRWorkMode(job.LocationType, job.IsRemote)},
				describe: func(ctx context.Context, r *Reader) (*Page, []string, *time.Time, error) {
					body, err := r.json(ctx, http.MethodGet, bambooHRAPI(company)+"/"+url.PathEscape(id)+"/detail", nil)
					if err != nil {
						return nil, nil, nil, err
					}
					page, posted, err := bambooHRDetail(body, company)
					if err != nil {
						return nil, nil, nil, fail(NoJobData, err)
					}
					page.ReadFrom = "ats_api"
					return page, nil, posted, nil
				}})
		}
		return postings, nil
	})
}

// bambooHR is the API call for a link to one opening.
func bambooHR(company, id string) *board {
	return &board{
		api: bambooHRAPI(company) + "/" + url.PathEscape(id) + "/detail",
		read: func(body []byte) (*Page, error) {
			page, _, err := bambooHRDetail(body, company)
			return page, err
		},
	}
}

func bambooHRDetail(body []byte, company string) (*Page, *time.Time, error) {
	var detail struct {
		Result struct {
			JobOpening struct {
				Name         string           `json:"jobOpeningName"`
				Location     bambooHRLocation `json:"location"`
				IsRemote     *bool            `json:"isRemote"`
				LocationType string           `json:"locationType"`
				Description  string           `json:"description"`
				DatePosted   string           `json:"datePosted"`
			} `json:"jobOpening"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &detail); err != nil || detail.Result.JobOpening.Name == "" {
		return nil, nil, firstError(err, errNoSuchJob)
	}
	job := detail.Result.JobOpening
	var posted *time.Time
	if at, err := time.Parse(time.DateOnly, job.DatePosted); err == nil {
		posted = &at
	}
	return &Page{Title: text(job.Name), Company: slugName(company), City: text(job.Location.City),
		Country: text(job.Location.AddressCountry), WorkMode: bambooHRWorkMode(job.LocationType, job.IsRemote),
		Text: strings.Join(descriptionLines(job.Description), "\n")}, posted, nil
}
