package fetch

import (
	"context"
	"encoding/json"
	"jobs/internal/extract"
	"net/url"
	"strings"
	"time"
)

// Workable serves each account's jobs, with their descriptions, from the widget its careers pages embed.
func workableAPI(account string) string {
	return "https://apply.workable.com/api/v1/widget/accounts/" + url.PathEscape(account) + "?details=true"
}

type workableAccount struct {
	Name string        `json:"name"`
	Jobs []workableJob `json:"jobs"`
}

type workableJob struct {
	Title         string `json:"title"`
	Shortcode     string `json:"shortcode"`
	URL           string `json:"url"`
	City          string `json:"city"`
	Country       string `json:"country"`
	Telecommuting bool   `json:"telecommuting"`
	PublishedOn   string `json:"published_on"`
	Description   string `json:"description"`
	Locations     []struct {
		City    string `json:"city"`
		Country string `json:"country"`
	} `json:"locations"`
}

func (j workableJob) page(account workableAccount, slug string) *Page {
	mode := ""
	if j.Telecommuting {
		mode = "remote"
	}
	return &Page{Title: text(j.Title), Company: extract.FirstOf(text(account.Name), slugName(slug)),
		City: text(j.City), Country: text(j.Country), WorkMode: mode,
		Text: strings.Join(descriptionLines(j.Description), "\n")}
}

// workableBoardOf reads the account from apply.workable.com/<account>; a short link (/j/<code>) names none.
func workableBoardOf(host string, parts []string, _ url.Values) (string, bool) {
	if host != "apply.workable.com" || len(parts) == 0 || parts[0] == "j" || parts[0] == "api" {
		return "", false
	}
	return parts[0], true
}

func (r *Reader) workableList(ctx context.Context, account string, _ Country) ([]Posting, bool, error) {
	return r.apiList(ctx, workableAPI(account), func(body []byte) ([]Posting, error) {
		var listing workableAccount
		if err := json.Unmarshal(body, &listing); err != nil {
			return nil, err
		}
		postings := []Posting{}
		for _, job := range listing.Jobs {
			places := []string{text(job.City), text(job.Country)}
			for _, where := range job.Locations {
				places = append(places, text(where.City), text(where.Country))
			}
			var posted *time.Time
			if at, err := time.Parse(time.DateOnly, job.PublishedOn); err == nil {
				posted = &at
			}
			postings = append(postings, Posting{ID: job.Shortcode, URL: job.URL, PostedAt: posted, Places: places,
				Page: *job.page(listing, account)})
		}
		return postings, nil
	})
}

// workable is the API call for a link to one job of an account.
func workable(account, shortcode string) *board {
	return &board{
		api: workableAPI(account),
		read: func(body []byte) (*Page, error) {
			var listing workableAccount
			if err := json.Unmarshal(body, &listing); err != nil {
				return nil, err
			}
			for _, job := range listing.Jobs {
				if strings.EqualFold(job.Shortcode, shortcode) {
					return job.page(listing, account), nil
				}
			}
			return nil, errNoSuchJob
		},
	}
}
