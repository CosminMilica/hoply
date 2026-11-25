package fetch

import (
	"encoding/json"
	"errors"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// board is a job board's public API call for one link and how to read its
// answer.
type board struct {
	api  string
	read func([]byte) (*Page, error)
}

var (
	digits          = regexp.MustCompile(`^\d+$`)
	leadingDigits   = regexp.MustCompile(`^(\d+)`)
	errNoSuchJob    = errors.New("the board does not list this job")
	greenhouseHosts = map[string]bool{"boards.greenhouse.io": true, "job-boards.greenhouse.io": true}
	leverAPIs       = map[string]string{"jobs.lever.co": "https://api.lever.co", "jobs.eu.lever.co": "https://api.eu.lever.co"}
)

// boardFor returns the API call for a known job board's link, or nil.
func boardFor(u *url.URL) *board {
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	parts := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	query := u.Query()
	switch {
	case greenhouseHosts[host] && len(parts) >= 3 && parts[1] == "jobs" && digits.MatchString(parts[2]):
		return greenhouse(parts[0], parts[2])
	case greenhouseHosts[host] && query.Get("for") != "" && digits.MatchString(query.Get("token")):
		return greenhouse(query.Get("for"), query.Get("token"))
	case digits.MatchString(query.Get("gh_jid")) && firstOf(query.Get("for"), query.Get("board")) != "":
		return greenhouse(firstOf(query.Get("for"), query.Get("board")), query.Get("gh_jid"))
	case leverAPIs[host] != "" && len(parts) >= 2:
		return lever(leverAPIs[host], parts[0], parts[1])
	case host == "jobs.ashbyhq.com" && len(parts) >= 2:
		return ashby(parts[0], parts[1])
	case host == "jobs.smartrecruiters.com" && len(parts) >= 2 && leadingDigits.MatchString(parts[1]):
		return smartRecruiters(parts[0], leadingDigits.FindString(parts[1]))
	case host == "apply.workable.com" && len(parts) >= 3 && parts[1] == "j":
		return workable(parts[0], parts[2])
	}
	return nil
}

func greenhouse(boardToken, id string) *board {
	return &board{
		api: "https://boards-api.greenhouse.io/v1/boards/" + url.PathEscape(boardToken) + "/jobs/" + id,
		read: func(body []byte) (*Page, error) {
			var job struct {
				Title       string `json:"title"`
				CompanyName string `json:"company_name"`
				Location    struct {
					Name string `json:"name"`
				} `json:"location"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(body, &job); err != nil || job.Title == "" {
				return nil, firstError(err, errNoSuchJob)
			}
			return &Page{Title: text(job.Title), Company: text(job.CompanyName), Location: text(job.Location.Name),
				Text: strings.Join(descriptionLines(job.Content), "\n")}, nil
		},
	}
}

func lever(api, site, id string) *board {
	return &board{
		api: api + "/v0/postings/" + url.PathEscape(site) + "/" + url.PathEscape(id),
		read: func(body []byte) (*Page, error) {
			var posting struct {
				Text       string `json:"text"`
				Categories struct {
					Location string `json:"location"`
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
			if err := json.Unmarshal(body, &posting); err != nil || posting.Text == "" {
				return nil, firstError(err, errNoSuchJob)
			}
			description := posting.Description
			for _, list := range posting.Lists {
				description += "<h3>" + html.EscapeString(list.Text) + "</h3><ul>" + list.Content + "</ul>"
			}
			description += posting.Additional
			return &Page{Title: text(posting.Text), Company: slugName(site), Location: text(posting.Categories.Location),
				Country: text(posting.Country), WorkMode: workplace(posting.WorkplaceType),
				Text: strings.Join(descriptionLines(description), "\n")}, nil
		},
	}
}

func ashby(boardName, id string) *board {
	return &board{
		api: "https://api.ashbyhq.com/posting-api/job-board/" + url.PathEscape(boardName),
		read: func(body []byte) (*Page, error) {
			var listing struct {
				Jobs []struct {
					ID            string `json:"id"`
					Title         string `json:"title"`
					Location      string `json:"location"`
					IsRemote      bool   `json:"isRemote"`
					WorkplaceType string `json:"workplaceType"`
					Address       struct {
						PostalAddress struct {
							AddressLocality string `json:"addressLocality"`
							AddressCountry  string `json:"addressCountry"`
						} `json:"postalAddress"`
					} `json:"address"`
					DescriptionHTML string `json:"descriptionHtml"`
				} `json:"jobs"`
			}
			if err := json.Unmarshal(body, &listing); err != nil {
				return nil, err
			}
			for _, job := range listing.Jobs {
				if strings.EqualFold(job.ID, id) {
					mode := workplace(job.WorkplaceType)
					if mode == "" && job.IsRemote {
						mode = "remote"
					}
					return &Page{Title: text(job.Title), Company: slugName(boardName), Location: text(job.Location),
						City: text(job.Address.PostalAddress.AddressLocality), Country: text(job.Address.PostalAddress.AddressCountry),
						WorkMode: mode, Text: strings.Join(descriptionLines(job.DescriptionHTML), "\n")}, nil
				}
			}
			return nil, errNoSuchJob
		},
	}
}

func smartRecruiters(company, id string) *board {
	return &board{
		api: "https://api.smartrecruiters.com/v1/companies/" + url.PathEscape(company) + "/postings/" + id,
		read: func(body []byte) (*Page, error) {
			type section struct {
				Text string `json:"text"`
			}
			var posting struct {
				Name    string `json:"name"`
				Company struct {
					Name string `json:"name"`
				} `json:"company"`
				Location struct {
					City    string `json:"city"`
					Country string `json:"country"`
					Remote  bool   `json:"remote"`
					Hybrid  bool   `json:"hybrid"`
				} `json:"location"`
				JobAd struct {
					Sections struct {
						CompanyDescription    section `json:"companyDescription"`
						JobDescription        section `json:"jobDescription"`
						Qualifications        section `json:"qualifications"`
						AdditionalInformation section `json:"additionalInformation"`
					} `json:"sections"`
				} `json:"jobAd"`
			}
			if err := json.Unmarshal(body, &posting); err != nil || posting.Name == "" {
				return nil, firstError(err, errNoSuchJob)
			}
			s := posting.JobAd.Sections
			// The job description leads, so its first lines become the about text;
			// the company description and additional information feed no field.
			description := s.JobDescription.Text + "<h3>Requirements</h3>" + s.Qualifications.Text +
				"<h3>What we offer</h3>" + s.AdditionalInformation.Text + "<h3>About us</h3>" + s.CompanyDescription.Text
			mode := ""
			switch {
			case posting.Location.Remote:
				mode = "remote"
			case posting.Location.Hybrid:
				mode = "hybrid"
			}
			return &Page{Title: text(posting.Name), Company: firstOf(text(posting.Company.Name), slugName(company)),
				City: text(posting.Location.City), Country: text(posting.Location.Country), WorkMode: mode,
				Text: strings.Join(descriptionLines(description), "\n")}, nil
		},
	}
}

func workable(account, shortcode string) *board {
	return &board{
		api: "https://apply.workable.com/api/v1/widget/accounts/" + url.PathEscape(account) + "?details=true",
		read: func(body []byte) (*Page, error) {
			var listing struct {
				Name string `json:"name"`
				Jobs []struct {
					Title         string `json:"title"`
					Shortcode     string `json:"shortcode"`
					City          string `json:"city"`
					Country       string `json:"country"`
					Telecommuting bool   `json:"telecommuting"`
					Description   string `json:"description"`
				} `json:"jobs"`
			}
			if err := json.Unmarshal(body, &listing); err != nil {
				return nil, err
			}
			for _, job := range listing.Jobs {
				if strings.EqualFold(job.Shortcode, shortcode) {
					mode := ""
					if job.Telecommuting {
						mode = "remote"
					}
					return &Page{Title: text(job.Title), Company: firstOf(text(listing.Name), slugName(account)),
						City: text(job.City), Country: text(job.Country), WorkMode: mode,
						Text: strings.Join(descriptionLines(job.Description), "\n")}, nil
				}
			}
			return nil, errNoSuchJob
		},
	}
}

// workplace maps a board's workplace type onto work_mode.
func workplace(value string) string {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "-", "")) {
	case "remote":
		return "remote"
	case "hybrid":
		return "hybrid"
	case "onsite":
		return "onsite"
	}
	return ""
}

// slugName turns a board slug into a company name: "northwind-cloud" becomes
// "Northwind Cloud". Boards whose API has no company name use it.
func slugName(slug string) string {
	words := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	for i, word := range words {
		r, size := utf8.DecodeRuneInString(word)
		words[i] = string(unicode.ToUpper(r)) + word[size:]
	}
	return strings.Join(words, " ")
}

func firstOf(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
