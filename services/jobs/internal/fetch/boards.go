package fetch

import (
	"encoding/json"
	"errors"
	"jobs/internal/extract"
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
	greenhouseHosts = map[string]bool{"boards.greenhouse.io": true, "job-boards.greenhouse.io": true, "boards.eu.greenhouse.io": true, "job-boards.eu.greenhouse.io": true}
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
	case digits.MatchString(query.Get("gh_jid")) && extract.FirstOf(query.Get("for"), query.Get("board")) != "":
		return greenhouse(extract.FirstOf(query.Get("for"), query.Get("board")), query.Get("gh_jid"))
	case leverAPIs[host] != "" && len(parts) >= 2:
		return lever(leverAPIs[host], parts[0], parts[1])
	case host == "jobs.ashbyhq.com" && len(parts) >= 2:
		return ashby(parts[0], parts[1])
	case host == "jobs.smartrecruiters.com" && len(parts) >= 2 && leadingDigits.MatchString(parts[1]):
		return smartRecruiters(parts[0], leadingDigits.FindString(parts[1]))
	case bambooHRCompany(host) != "" && len(parts) >= 2 && parts[0] == "careers" && digits.MatchString(parts[1]):
		return bambooHR(bambooHRCompany(host), parts[1])
	case recruiteeCompany(host) != "" && len(parts) >= 2 && parts[0] == "o":
		return recruitee(recruiteeCompany(host), parts[1])
	case host == "apply.workable.com" && len(parts) >= 3 && parts[1] == "j":
		return workable(parts[0], parts[2])
	}
	return nil
}

func greenhouse(boardToken, id string) *board {
	return &board{
		api: "https://boards-api.greenhouse.io/v1/boards/" + url.PathEscape(boardToken) + "/jobs/" + id,
		read: func(body []byte) (*Page, error) {
			var job greenhouseJob
			if err := json.Unmarshal(body, &job); err != nil || job.Title == "" {
				return nil, firstError(err, errNoSuchJob)
			}
			return job.page(), nil
		},
	}
}

func lever(api, site, id string) *board {
	return &board{
		api: api + "/v0/postings/" + url.PathEscape(site) + "/" + url.PathEscape(id),
		read: func(body []byte) (*Page, error) {
			var posting leverPosting
			if err := json.Unmarshal(body, &posting); err != nil || posting.Text == "" {
				return nil, firstError(err, errNoSuchJob)
			}
			return posting.page(site), nil
		},
	}
}

func ashby(boardName, id string) *board {
	return &board{
		api: "https://api.ashbyhq.com/posting-api/job-board/" + url.PathEscape(boardName),
		read: func(body []byte) (*Page, error) {
			var listing ashbyBoard
			if err := json.Unmarshal(body, &listing); err != nil {
				return nil, err
			}
			for _, job := range listing.Jobs {
				if strings.EqualFold(job.ID, id) {
					return job.page(boardName), nil
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
			return &Page{Title: text(posting.Name), Company: extract.FirstOf(text(posting.Company.Name), slugName(company)),
				City: text(posting.Location.City), Country: text(posting.Location.Country), WorkMode: mode,
				Text: strings.Join(descriptionLines(description), "\n")}, nil
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

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

var workdayLocale = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)

// BoardOf returns the platform and board a job link belongs to, when List can
// read that board: "https://jobs.lever.co/northwind/aa11" is lever,
// "northwind".
func BoardOf(link string) (platform, board string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", "", false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	parts := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	for _, p := range platforms {
		if board, ok := p.boardOf(host, parts, u.Query()); ok {
			if CheckBoard(p.name, board) != nil {
				return "", "", false
			}
			return p.name, board, true
		}
	}
	return "", "", false
}

func greenhouseBoardOf(host string, parts []string, query url.Values) (string, bool) {
	if greenhouseHosts[host] && len(parts) > 0 && parts[0] != "embed" {
		return parts[0], true
	}
	if greenhouseHosts[host] || query.Get("gh_jid") != "" {
		return extract.FirstOf(query.Get("for"), query.Get("board")), true
	}
	return "", false
}

func workdayBoardOf(host string, parts []string, _ url.Values) (string, bool) {
	if !workdayHost.MatchString(host) {
		return "", false
	}
	first := ""
	if len(parts) > 0 {
		first = parts[0]
	}
	if len(parts) > 1 && workdayLocale.MatchString(first) {
		first = parts[1]
	}
	return host + "/" + first, true
}
