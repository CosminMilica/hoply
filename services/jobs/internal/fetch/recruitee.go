package fetch

import (
	"context"
	"encoding/json"
	"jobs/internal/extract"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Recruitee serves each company's careers site from <company>.recruitee.com: /api/offers/ lists every published
// offer with its description and requirements.
const recruiteeHost = ".recruitee.com"

func recruiteeAPI(company string) string {
	return "https://" + url.PathEscape(company) + recruiteeHost + "/api/offers/"
}

// recruiteeCompany is the company a Recruitee host belongs to, its subdomain, or "".
func recruiteeCompany(host string) string {
	company, ok := strings.CutSuffix(host, recruiteeHost)
	if !ok || company == "" || strings.Contains(company, ".") {
		return ""
	}
	return company
}

func recruiteeBoardOf(host string, _ []string, _ url.Values) (string, bool) {
	company := recruiteeCompany(host)
	return company, company != ""
}

type recruiteeOffer struct {
	ID           int64  `json:"id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	CompanyName  string `json:"company_name"`
	CareersURL   string `json:"careers_url"`
	Location     string `json:"location"`
	City         string `json:"city"`
	Country      string `json:"country"`
	Remote       bool   `json:"remote"`
	Hybrid       bool   `json:"hybrid"`
	PublishedAt  string `json:"published_at"`
	Description  string `json:"description"`
	Requirements string `json:"requirements"`
	Locations    []struct {
		City    string `json:"city"`
		Country string `json:"country"`
	} `json:"locations"`
}

type recruiteeOffers struct {
	Offers []recruiteeOffer `json:"offers"`
}

func (o recruiteeOffer) page(company string) *Page {
	mode := ""
	switch {
	case o.Remote:
		mode = "remote"
	case o.Hybrid:
		mode = "hybrid"
	}
	description := o.Description
	if strings.TrimSpace(o.Requirements) != "" {
		description += "<h3>Requirements</h3>" + o.Requirements
	}
	return &Page{Title: text(o.Title), Company: extract.FirstOf(text(o.CompanyName), slugName(company)),
		Location: text(o.Location), City: text(o.City), Country: text(o.Country), WorkMode: mode,
		Text: strings.Join(descriptionLines(description), "\n")}
}

func (r *Reader) recruiteeList(ctx context.Context, company string, _ Country) ([]Posting, bool, error) {
	return r.apiList(ctx, recruiteeAPI(company), func(body []byte) ([]Posting, error) {
		var listing recruiteeOffers
		if err := json.Unmarshal(body, &listing); err != nil {
			return nil, err
		}
		postings := []Posting{}
		for _, offer := range listing.Offers {
			places := []string{text(offer.Location), text(offer.City), text(offer.Country)}
			for _, where := range offer.Locations {
				places = append(places, text(where.City), text(where.Country))
			}
			var posted *time.Time
			if at, err := time.Parse("2006-01-02 15:04:05 MST", offer.PublishedAt); err == nil {
				at = at.UTC()
				posted = &at
			}
			postings = append(postings, Posting{ID: strconv.FormatInt(offer.ID, 10), URL: offer.CareersURL, PostedAt: posted,
				Places: places, Page: *offer.page(company)})
		}
		return postings, nil
	})
}

// recruitee is the API call for a link to one offer, <company>.recruitee.com/o/<slug>.
func recruitee(company, slug string) *board {
	return &board{
		api: recruiteeAPI(company),
		read: func(body []byte) (*Page, error) {
			var listing recruiteeOffers
			if err := json.Unmarshal(body, &listing); err != nil {
				return nil, err
			}
			for _, offer := range listing.Offers {
				if offer.Slug == slug {
					return offer.page(company), nil
				}
			}
			return nil, errNoSuchJob
		},
	}
}
