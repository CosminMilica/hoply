package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Platforms are the job board platforms List reads. A sitemap board is a
// career site's sitemap; its job pages are read like a link.
var Platforms = []string{"greenhouse", "lever", "lever-eu", "ashby", "workday", "smartrecruiters", "sitemap"}

// Country is the market a list keeps to on the platforms that filter by
// country (Workday, SmartRecruiters): its ISO code, its English name, and In,
// which tells whether a place such as "Bucharest - Dacia One" is in it.
type Country struct {
	Code string
	Name string
	In   func(place string) bool
}

// has reports whether a place is in the country: by In, else by its name.
func (c Country) has(place string) bool {
	if c.In != nil {
		return c.In(place)
	}
	return strings.Contains(strings.ToLower(place), strings.ToLower(c.Name))
}

// Posting is one job a company's board lists. Places are every location the
// board gives the job. InCountry is set when the platform itself kept the list
// to the country. A posting whose list has no description is read by Describe.
type Posting struct {
	ID        string
	URL       string
	PostedAt  *time.Time
	Places    []string
	InCountry bool
	Page      Page
	describe  func(ctx context.Context, r *Reader) (*Page, []string, *time.Time, error)
}

// CheckBoard reports whether board is a valid board name on platform.
func CheckBoard(platform, board string) error {
	switch platform {
	case "workday":
		_, _, err := workdaySite(board)
		return err
	case "sitemap":
		if u, err := url.Parse(board); err != nil || u.Scheme != "https" || u.Hostname() == "" {
			return fmt.Errorf("a sitemap board is an https link, got %q", board)
		}
		return nil
	case "greenhouse", "lever", "lever-eu", "ashby", "smartrecruiters":
		if board == "" || strings.ContainsAny(board, "/?# ") {
			return fmt.Errorf("a %s board is a name without slashes, got %q", platform, board)
		}
		return nil
	}
	return fmt.Errorf("unknown platform %q", platform)
}

// List reads every job a company's board lists on platform; Workday and
// SmartRecruiters keep to country. complete is false when a board had more jobs
// than a list reads, so the jobs missing from it may still be open.
func (r *Reader) List(ctx context.Context, platform, board string, country Country) (postings []Posting, complete bool, err error) {
	if err := CheckBoard(platform, board); err != nil {
		return nil, false, err
	}
	switch platform {
	case "workday":
		return r.workday(ctx, board, country)
	case "smartrecruiters":
		return r.smartRecruitersList(ctx, board, country)
	case "sitemap":
		return r.sitemap(ctx, board)
	}
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
	default:
		api, read = "https://api.ashbyhq.com/posting-api/job-board/"+url.PathEscape(board), func(body []byte) ([]Posting, error) { return ashbyList(body, board) }
	}
	body, err := r.json(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, false, err
	}
	if postings, err = read(body); err != nil {
		return nil, false, fail(NoJobData, err)
	}
	for i := range postings {
		postings[i].Page.ReadFrom = "ats_api"
	}
	return postings, true, nil
}

// Describe reads a posting's description when its list had none; it does
// nothing for a posting that has it.
func (r *Reader) Describe(ctx context.Context, p *Posting) error {
	if p.describe == nil {
		return nil
	}
	page, places, posted, err := p.describe(ctx, r)
	if err != nil {
		return err
	}
	page.Title, page.Company = firstOf(page.Title, p.Page.Title), firstOf(page.Company, p.Page.Company)
	page.City, page.Country, page.WorkMode = firstOf(page.City, p.Page.City), firstOf(page.Country, p.Page.Country), firstOf(page.WorkMode, p.Page.WorkMode)
	p.Page, p.Places, p.describe = *page, append(p.Places, places...), nil
	if posted != nil {
		p.PostedAt = posted
	}
	return nil
}

// json calls a board API and returns its JSON answer.
func (r *Reader) json(ctx context.Context, method, api string, body any) ([]byte, error) {
	target, err := url.Parse(api)
	if err != nil {
		return nil, fail(FetchFailed, err)
	}
	var payload []byte
	if body != nil {
		if payload, err = json.Marshal(body); err != nil {
			return nil, fail(FetchFailed, err)
		}
	}
	answer, _, err := r.send(ctx, method, target, payload, r.opts.MaxAPIBytes, "application/json")
	return answer, err
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
