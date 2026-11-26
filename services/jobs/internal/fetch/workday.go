package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	workdayPage = 20
	// maxWorkdayJobs caps one board's list; a country rarely has more.
	maxWorkdayJobs = 400
)

var (
	workdayHost = regexp.MustCompile(`^[a-z0-9-]+\.wd\d+\.myworkdayjobs\.com$`)
	daysAgo     = regexp.MustCompile(`(?i)posted (\d+) days? ago`)
)

// workdaySite splits a Workday board, "{tenant}.wdN.myworkdayjobs.com/{site}"
// as in its career site's links, into the site's API and its career site.
func workdaySite(board string) (api, site string, err error) {
	host, name, ok := strings.Cut(board, "/")
	if !ok || !workdayHost.MatchString(host) || name == "" || strings.ContainsAny(name, "/?# ") {
		return "", "", fmt.Errorf("a workday board is {tenant}.wdN.myworkdayjobs.com/{site}, got %q", board)
	}
	tenant, _, _ := strings.Cut(host, ".")
	return "https://" + host + "/wday/cxs/" + tenant + "/" + name, "https://" + host + "/" + name, nil
}

type workdayFacet struct {
	FacetParameter string         `json:"facetParameter"`
	Descriptor     string         `json:"descriptor"`
	ID             string         `json:"id"`
	Values         []workdayFacet `json:"values"`
}

type workdayList struct {
	Total       int `json:"total"`
	JobPostings []struct {
		Title         string `json:"title"`
		ExternalPath  string `json:"externalPath"`
		LocationsText string `json:"locationsText"`
		PostedOn      string `json:"postedOn"`
	} `json:"jobPostings"`
	Facets []workdayFacet `json:"facets"`
}

// countryFacet finds the facet value that keeps a site's jobs to a country:
// the value named like the country under a facet whose parameter names a
// country. Sites name that facet differently.
func countryFacet(facets []workdayFacet, name, parent string) (param, id string) {
	for _, facet := range facets {
		if facet.FacetParameter != "" {
			if param, id := countryFacet(facet.Values, name, facet.FacetParameter); id != "" {
				return param, id
			}
			continue
		}
		if strings.Contains(strings.ToLower(parent), "country") && strings.EqualFold(strings.TrimSpace(facet.Descriptor), name) && facet.ID != "" {
			return parent, facet.ID
		}
	}
	return "", ""
}

// placeFacet finds, when a site has no country facet, the values of its first
// location facet that are in the country: "Bucharest - Dacia One" and
// "Cluj-Napoca, Romania" for Romania.
func placeFacet(facets []workdayFacet, country Country) (param string, ids []string) {
	for _, facet := range facets {
		if facet.FacetParameter == "" {
			continue
		}
		if strings.Contains(strings.ToLower(facet.FacetParameter), "location") {
			for _, value := range facet.Values {
				if value.FacetParameter == "" && value.ID != "" && country.has(value.Descriptor) {
					ids = append(ids, value.ID)
				}
			}
			if len(ids) > 0 {
				return facet.FacetParameter, ids
			}
		}
		if param, ids := placeFacet(facet.Values, country); len(ids) > 0 {
			return param, ids
		}
	}
	return "", nil
}

// workday lists a Workday site's jobs in the country, a page at a time, by its
// country facet or else by its locations in the country; a site with neither
// lists none. complete is false when the list stopped at maxWorkdayJobs.
func (r *Reader) workday(ctx context.Context, board string, country Country) (postings []Posting, complete bool, err error) {
	api, site, _ := workdaySite(board)
	search := func(facets map[string][]string, offset int) (*workdayList, error) {
		body, err := r.json(ctx, http.MethodPost, api+"/jobs", map[string]any{"appliedFacets": facets, "limit": workdayPage, "offset": offset, "searchText": ""})
		if err != nil {
			return nil, err
		}
		var list workdayList
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fail(NoJobData, err)
		}
		return &list, nil
	}
	first, err := search(map[string][]string{}, 0)
	if err != nil {
		return nil, false, err
	}
	param, id := countryFacet(first.Facets, country.Name, "")
	ids := []string{id}
	if id == "" {
		param, ids = placeFacet(first.Facets, country)
	}
	postings = []Posting{}
	if len(ids) == 0 {
		return postings, true, nil
	}
	total := 0
	for offset := 0; ; offset += workdayPage {
		if offset >= maxWorkdayJobs {
			return postings, false, nil
		}
		list, err := search(map[string][]string{param: ids}, offset)
		if err != nil {
			return nil, false, err
		}
		if offset == 0 {
			total = list.Total // later pages answer 0
		}
		for _, job := range list.JobPostings {
			path := job.ExternalPath
			postings = append(postings, Posting{ID: path, URL: site + path, PostedAt: postedOn(job.PostedOn, time.Now()),
				Places: []string{text(job.LocationsText)}, InCountry: true, Page: Page{Title: text(job.Title)},
				describe: func(ctx context.Context, r *Reader) (*Page, []string, *time.Time, error) {
					return r.workdayJob(ctx, api+path)
				}})
		}
		if len(list.JobPostings) < workdayPage || offset+workdayPage >= total {
			return postings, true, nil
		}
	}
}

// workdayJob reads one job of a Workday site.
func (r *Reader) workdayJob(ctx context.Context, api string) (*Page, []string, *time.Time, error) {
	body, err := r.json(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	var job struct {
		Info struct {
			Title               string   `json:"title"`
			JobDescription      string   `json:"jobDescription"`
			Location            string   `json:"location"`
			AdditionalLocations []string `json:"additionalLocations"`
			StartDate           string   `json:"startDate"`
			RemoteType          string   `json:"remoteType"`
			Country             struct {
				Descriptor string `json:"descriptor"`
			} `json:"country"`
		} `json:"jobPostingInfo"`
		HiringOrganization struct {
			Name string `json:"name"`
		} `json:"hiringOrganization"`
	}
	if err := json.Unmarshal(body, &job); err != nil || job.Info.Title == "" {
		return nil, nil, nil, fail(NoJobData, firstError(err, errNoSuchJob))
	}
	info := job.Info
	page := &Page{ReadFrom: "ats_api", Title: text(info.Title), Company: text(job.HiringOrganization.Name), Location: text(info.Location),
		Country: text(info.Country.Descriptor), WorkMode: workplace(info.RemoteType), Text: strings.Join(descriptionLines(info.JobDescription), "\n")}
	places := append([]string{page.Location, page.Country}, info.AdditionalLocations...)
	var posted *time.Time
	if at, err := time.Parse(time.DateOnly, info.StartDate); err == nil {
		posted = &at
	}
	return page, places, posted, nil
}

// postedOn reads Workday's "Posted Today", "Posted Yesterday" and "Posted 3
// Days Ago"; "Posted 30+ Days Ago" and anything else is nil.
func postedOn(value string, now time.Time) *time.Time {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	switch folded := strings.ToLower(strings.TrimSpace(value)); {
	case folded == "posted today":
		return &day
	case folded == "posted yesterday":
		day = day.AddDate(0, 0, -1)
		return &day
	case daysAgo.MatchString(folded):
		n, _ := strconv.Atoi(daysAgo.FindStringSubmatch(folded)[1])
		day = day.AddDate(0, 0, -n)
		return &day
	}
	return nil
}
