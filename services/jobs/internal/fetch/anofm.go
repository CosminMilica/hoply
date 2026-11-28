package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ANOFM, Romania's national employment agency, publishes the vacancies employers register with it on its job
// matching portal: the public search reads POST /api/entity/vw_public_job_posting a page at a time, and each job has a
// page, /app/module/mediere/job/{id}, with its responsibilities. Most of its employers are small companies with no job
// board of their own. A board is a county, by its code (IS for Iași); its jobs' employer is each job's own.
const (
	anofmSite = "https://mediere.anofm.ro"
	anofmPage = 100
	// maxANOFMJobs bounds one county's list: Bucharest, the largest, lists about 2,000.
	maxANOFMJobs = 5000
)

// anofmCounties are the counties' codes and ANOFM's ids for them.
var anofmCounties = map[string]int{
	"AB": 1, "AG": 4, "AR": 3, "B": 2, "BC": 5, "BH": 6, "BN": 7, "BR": 10, "BT": 8, "BV": 9, "BZ": 11, "CJ": 13,
	"CL": 40, "CS": 12, "CT": 14, "CV": 15, "DB": 16, "DJ": 17, "GJ": 19, "GL": 18, "GR": 41, "HD": 21, "HR": 20,
	"IF": 42, "IL": 22, "IS": 23, "MH": 25, "MM": 24, "MS": 26, "NT": 27, "OT": 28, "PH": 29, "SB": 32, "SJ": 31,
	"SM": 30, "SV": 33, "TL": 36, "TM": 35, "TR": 34, "VL": 38, "VN": 39, "VS": 37,
}

func anofmCheck(board string) error {
	if _, ok := anofmCounties[board]; !ok {
		return fmt.Errorf("an anofm board is a county code such as IS, got %q", board)
	}
	return nil
}

type anofmJob struct {
	ID           int64   `json:"id"`
	CreatedAt    string  `json:"created_at"`
	Occupation   string  `json:"occupation"`
	EmployerName string  `json:"employer_name"`
	Locality     string  `json:"address_locality_name"`
	WorkRegime   string  `json:"work_regime_name"`
	Experience   string  `json:"professional_experience_name"`
	Description  *string `json:"description"`
}

type anofmList struct {
	Rows  []anofmJob `json:"rows"`
	Total int        `json:"total"`
}

// anofmMode is a job's work mode from its work regime: Sediu, Telemuncă, or one of the hybrids.
func anofmMode(regime string) string {
	folded := strings.ToLower(regime)
	switch {
	case strings.HasPrefix(folded, "hibrid"):
		return "hybrid"
	case strings.HasPrefix(folded, "telemunc"), strings.Contains(folded, "domiciliu"):
		return "remote"
	case strings.HasPrefix(folded, "sediu"):
		return "onsite"
	}
	return ""
}

// anofmLevel is a job's level from the experience it asks: none or under 3 years junior, 3 to 5 mid, more senior.
func anofmLevel(experience string) string {
	folded := strings.ToLower(experience)
	switch {
	case strings.HasPrefix(folded, "fără"), strings.HasPrefix(folded, "fara"), strings.Contains(folded, "puțină"), strings.Contains(folded, "putina"):
		return "junior"
	case strings.Contains(folded, "medie"):
		return "mid"
	case strings.HasPrefix(folded, "avansat"), strings.HasPrefix(folded, "experimentat"):
		return "senior"
	}
	return ""
}

// sentenceCase writes an occupation the way a title reads: "CONTABIL SEF" is "Contabil sef".
func sentenceCase(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return ""
	}
	lower := []rune(strings.ToLower(value))
	return strings.ToUpper(string(lower[0])) + string(lower[1:])
}

func (r *Reader) anofmList(ctx context.Context, county string, _ Country) (postings []Posting, complete bool, err error) {
	id := anofmCounties[county]
	postings = []Posting{}
	for page := 1; ; page++ {
		if (page-1)*anofmPage >= maxANOFMJobs {
			return postings, false, nil
		}
		body, err := r.json(ctx, http.MethodPost, anofmSite+"/api/entity/vw_public_job_posting",
			map[string]any{"current": page, "rowCount": anofmPage, "searchPhrase": "", "county_id": id})
		if err != nil {
			return nil, false, err
		}
		var list anofmList
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, false, fail(NoJobData, err)
		}
		for _, job := range list.Rows {
			postings = append(postings, job.posting())
		}
		if len(list.Rows) < anofmPage || page*anofmPage >= list.Total {
			return postings, true, nil
		}
	}
}

func (job anofmJob) posting() Posting {
	id := strconv.FormatInt(job.ID, 10)
	link := anofmSite + "/app/module/mediere/job/" + id
	var posted *time.Time
	if at, err := time.Parse(time.DateTime, job.CreatedAt); err == nil {
		posted = &at
	}
	posting := Posting{ID: id, URL: link, PostedAt: posted, Places: []string{strings.ReplaceAll(text(job.Locality), " > ", ", ")},
		InCountry: true, Employer: text(job.EmployerName),
		Page: Page{Title: sentenceCase(text(job.Occupation)), Company: text(job.EmployerName), Country: "Romania",
			WorkMode: anofmMode(job.WorkRegime), Level: anofmLevel(job.Experience), ReadFrom: "ats_api"}}
	posting.describe = func(ctx context.Context, r *Reader) (*Page, []string, *time.Time, error) {
		return r.anofmJob(ctx, link, posting.Page)
	}
	return posting
}

// anofmFields are the job page's fields a job keeps, in the order its text lists them; the contact person, their email
// and phone and the address stay on ANOFM's page.
var anofmFields = []string{
	"Responsabilități", "Condiții de muncă", "Nivel de educație", "Experiență profesională", "Tip contract", "Tip normă",
	"Regim de muncă", "Beneficii",
}

// anofmJob reads one job's page: its responsibilities, then what it asks and offers, as lines.
func (r *Reader) anofmJob(ctx context.Context, link string, listed Page) (*Page, []string, *time.Time, error) {
	target, err := url.Parse(link)
	if err != nil {
		return nil, nil, nil, fail(FetchFailed, err)
	}
	body, _, err := r.get(ctx, target, "text/html", r.opts.MaxBytes)
	if err != nil {
		return nil, nil, nil, err
	}
	fields, err := anofmFieldsOf(body)
	if err != nil || fields["ID Ofertă"] == "" {
		return nil, nil, nil, fail(NoJobData, firstError(err, errNoSuchJob))
	}
	lines := []string{}
	for _, name := range anofmFields {
		value := strings.TrimSpace(fields[name])
		switch {
		case value == "":
		case name == "Responsabilități":
			lines = append(lines, "Responsabilități")
			for _, line := range strings.Split(value, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					lines = append(lines, "- "+line)
				}
			}
		default:
			lines = append(lines, name+": "+value)
		}
	}
	page := listed
	page.Text = strings.Join(lines, "\n")
	return &page, nil, nil, nil
}

// anofmFieldsOf reads a job page's fields: each span.info-label with the span.info-value after it, a value of several
// badges as a comma-separated list.
func anofmFieldsOf(page []byte) (map[string]string, error) {
	root, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "info-label") {
			label := strings.TrimSuffix(strings.TrimSpace(nodeText(n)), ":")
			for value := n.NextSibling; value != nil; value = value.NextSibling {
				if value.Type == html.ElementNode {
					if value.Data == "span" && hasClass(value, "info-value") {
						fields[label] = valueText(value)
					}
					break
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return fields, nil
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" && strings.Contains(" "+attr.Val+" ", " "+class+" ") {
			return true
		}
	}
	return false
}

func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(nodeText(child))
	}
	return b.String()
}

// valueText is a value's text with its lines kept; a value made of badges is their texts, comma-separated.
func valueText(n *html.Node) string {
	badges := []string{}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "span" {
			if badge := strings.TrimSpace(nodeText(child)); badge != "" {
				badges = append(badges, badge)
			}
		}
	}
	if len(badges) > 0 {
		return strings.Join(badges, ", ")
	}
	lines := []string{}
	for _, line := range strings.Split(nodeText(n), "\n") {
		if line = strings.Join(strings.Fields(line), " "); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
