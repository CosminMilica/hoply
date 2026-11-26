package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Platforms are the job board platforms List reads.
var Platforms = []string{"greenhouse", "lever", "lever-eu", "ashby"}

// Posting is one job a company's board lists. Places are every location the
// board gives the job, the page's own first.
type Posting struct {
	ID       string
	URL      string
	PostedAt *time.Time
	Places   []string
	Page     Page
}

// List reads every job a company's board lists on platform, in one call to
// the platform's public API.
func (r *Reader) List(ctx context.Context, platform, board string) ([]Posting, error) {
	var api string
	var read func([]byte) ([]Posting, error)
	switch platform {
	case "greenhouse":
		api, read = "https://boards-api.greenhouse.io/v1/boards/"+url.PathEscape(board)+"/jobs?content=true", greenhouseList
	case "lever", "lever-eu":
		host := leverAPIs["jobs.lever.co"]
		if platform == "lever-eu" {
			host = leverAPIs["jobs.eu.lever.co"]
		}
		api, read = host+"/v0/postings/"+url.PathEscape(board)+"?mode=json", func(body []byte) ([]Posting, error) { return leverList(body, board) }
	case "ashby":
		api, read = "https://api.ashbyhq.com/posting-api/job-board/"+url.PathEscape(board), func(body []byte) ([]Posting, error) { return ashbyList(body, board) }
	default:
		return nil, fmt.Errorf("unknown platform %q", platform)
	}
	target, err := url.Parse(api)
	if err != nil {
		return nil, fail(FetchFailed, err)
	}
	body, _, err := r.get(ctx, target, "application/json", r.opts.MaxAPIBytes)
	if err != nil {
		return nil, err
	}
	postings, err := read(body)
	if err != nil {
		return nil, fail(NoJobData, err)
	}
	for i := range postings {
		postings[i].Page.ReadFrom = "ats_api"
	}
	return postings, nil
}

type greenhouseJob struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	CompanyName string `json:"company_name"`
	Location    struct {
		Name string `json:"name"`
	} `json:"location"`
	Offices []struct {
		Location string `json:"location"`
	} `json:"offices"`
	Content        string `json:"content"`
	AbsoluteURL    string `json:"absolute_url"`
	FirstPublished string `json:"first_published"`
	UpdatedAt      string `json:"updated_at"`
}

// page reads the job; Greenhouse escapes the content's HTML once more.
func (j greenhouseJob) page() *Page {
	return &Page{Title: text(j.Title), Company: text(j.CompanyName), Location: text(j.Location.Name),
		Text: strings.Join(descriptionLines(j.Content), "\n")}
}

func greenhouseList(body []byte) ([]Posting, error) {
	var listing struct {
		Jobs []greenhouseJob `json:"jobs"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, err
	}
	postings := []Posting{}
	for _, job := range listing.Jobs {
		places := []string{text(job.Location.Name)}
		for _, office := range job.Offices {
			places = append(places, text(office.Location))
		}
		postings = append(postings, Posting{ID: strconv.FormatInt(job.ID, 10), URL: job.AbsoluteURL,
			PostedAt: timeOf(firstOf(job.FirstPublished, job.UpdatedAt)), Places: places, Page: *job.page()})
	}
	return postings, nil
}

type leverPosting struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	HostedURL  string `json:"hostedUrl"`
	CreatedAt  int64  `json:"createdAt"`
	Categories struct {
		Location     string   `json:"location"`
		AllLocations []string `json:"allLocations"`
	} `json:"categories"`
	WorkplaceType string `json:"workplaceType"`
	Country       string `json:"country"`
	Description   string `json:"description"`
	Lists         []struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	} `json:"lists"`
	Additional string `json:"additional"`
}

func (p leverPosting) page(site string) *Page {
	description := p.Description
	for _, list := range p.Lists {
		description += "<h3>" + html.EscapeString(list.Text) + "</h3><ul>" + list.Content + "</ul>"
	}
	description += p.Additional
	return &Page{Title: text(p.Text), Company: slugName(site), Location: text(p.Categories.Location),
		Country: text(p.Country), WorkMode: workplace(p.WorkplaceType),
		Text: strings.Join(descriptionLines(description), "\n")}
}

func leverList(body []byte, site string) ([]Posting, error) {
	var listing []leverPosting
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, err
	}
	postings := []Posting{}
	for _, posting := range listing {
		var posted *time.Time
		if posting.CreatedAt > 0 {
			at := time.UnixMilli(posting.CreatedAt).UTC()
			posted = &at
		}
		places := append([]string{text(posting.Categories.Location)}, posting.Categories.AllLocations...)
		postings = append(postings, Posting{ID: posting.ID, URL: posting.HostedURL, PostedAt: posted, Places: places, Page: *posting.page(site)})
	}
	return postings, nil
}

type ashbyJob struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	JobURL             string `json:"jobUrl"`
	PublishedAt        string `json:"publishedAt"`
	IsListed           *bool  `json:"isListed"`
	Location           string `json:"location"`
	SecondaryLocations []struct {
		Location string `json:"location"`
	} `json:"secondaryLocations"`
	IsRemote      bool   `json:"isRemote"`
	WorkplaceType string `json:"workplaceType"`
	Address       struct {
		PostalAddress struct {
			AddressLocality string `json:"addressLocality"`
			AddressCountry  string `json:"addressCountry"`
		} `json:"postalAddress"`
	} `json:"address"`
	DescriptionHTML string `json:"descriptionHtml"`
}

type ashbyBoard struct {
	Jobs []ashbyJob `json:"jobs"`
}

func (j ashbyJob) page(boardName string) *Page {
	mode := workplace(j.WorkplaceType)
	if mode == "" && j.IsRemote {
		mode = "remote"
	}
	return &Page{Title: text(j.Title), Company: slugName(boardName), Location: text(j.Location),
		City: text(j.Address.PostalAddress.AddressLocality), Country: text(j.Address.PostalAddress.AddressCountry),
		WorkMode: mode, Text: strings.Join(descriptionLines(j.DescriptionHTML), "\n")}
}

func ashbyList(body []byte, boardName string) ([]Posting, error) {
	var listing ashbyBoard
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, err
	}
	postings := []Posting{}
	for _, job := range listing.Jobs {
		if job.IsListed != nil && !*job.IsListed {
			continue
		}
		places := []string{text(job.Location), text(job.Address.PostalAddress.AddressCountry)}
		for _, other := range job.SecondaryLocations {
			places = append(places, text(other.Location))
		}
		postings = append(postings, Posting{ID: job.ID, URL: job.JobURL, PostedAt: timeOf(job.PublishedAt), Places: places, Page: *job.page(boardName)})
	}
	return postings, nil
}

// timeOf parses an RFC 3339 time, or returns nil.
func timeOf(value string) *time.Time {
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	at = at.UTC()
	return &at
}
