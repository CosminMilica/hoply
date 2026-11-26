package fetch

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	smartRecruitersPage = 100
	// maxSmartRecruitersJobs and maxSitemapJobs cap one board's list.
	maxSmartRecruitersJobs = 500
	maxSitemapJobs         = 5000
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

// sitemap lists the job pages of a career site's sitemap: the links whose path
// has a /job/ or /jobs/ segment. Their title and places come from the link's
// words until Describe reads the page. complete is false when the list stopped
// at maxSitemapJobs.
func (r *Reader) sitemap(ctx context.Context, board string) (postings []Posting, complete bool, err error) {
	target, _ := url.Parse(board)
	body, _, err := r.send(ctx, http.MethodGet, target, nil, r.opts.MaxAPIBytes, "text/xml", "application/xml")
	if err != nil {
		return nil, false, err
	}
	var set struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(body, &set); err != nil {
		return nil, false, fail(NoJobData, err)
	}
	postings = []Posting{}
	for _, entry := range set.URLs {
		link, err := url.Parse(strings.TrimSpace(entry.Loc))
		if err != nil || link.Hostname() != target.Hostname() || !jobPath(link.Path) {
			continue
		}
		words := strings.FieldsFunc(link.Path, func(r rune) bool { return r == '/' })
		title := ""
		for _, word := range words {
			if strings.Count(word, "-") > strings.Count(title, "-") {
				title = word
			}
		}
		page := link.String()
		postings = append(postings, Posting{ID: link.Path, URL: page, Places: []string{strings.Join(words, " ")},
			Page: Page{Title: strings.ReplaceAll(title, "-", " ")},
			describe: func(ctx context.Context, r *Reader) (*Page, []string, *time.Time, error) {
				read, err := r.Read(ctx, page)
				if err != nil {
					return nil, nil, nil, err
				}
				return read, []string{read.Location, read.City, read.Country}, nil, nil
			}})
		if len(postings) == maxSitemapJobs {
			return postings, false, nil
		}
	}
	return postings, true, nil
}

func jobPath(path string) bool {
	for _, segment := range strings.Split(strings.ToLower(path), "/") {
		if segment == "job" || segment == "jobs" {
			return true
		}
	}
	return false
}
