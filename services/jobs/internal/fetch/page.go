package fetch

import (
	"bytes"
	"encoding/json"
	"html"
	"jobs/internal/extract"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// readPage reads the JobPosting JSON-LD of a page, else its main text.
func readPage(body []byte, final *url.URL) (*Page, error) {
	doc, err := xhtml.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fail(FetchFailed, err)
	}
	if posting := findPosting(doc); posting != nil {
		return posting, nil
	}
	main := mainNode(doc)
	content := strings.Join(lines(main, true), "\n")
	if loginPage(main, final) {
		return nil, fail(LoginRequired, nil)
	}
	if utf8.RuneCountInString(content) < minMainText {
		return nil, fail(NoJobData, nil)
	}
	title, company := pageTitle(doc, main)
	return &Page{ReadFrom: "page_text", Text: content, Title: title, Company: company}, nil
}

// pageTitle reads the job title from the first <h1> (in the main content, then
// anywhere), else og:title, else <title>; and the company from og:site_name,
// else a <title> of the form "… at {Company}".
func pageTitle(doc, main *xhtml.Node) (title, company string) {
	documentTitle := ""
	var h1 string
	meta := map[string]string{}
	walk(doc, func(n *xhtml.Node) bool {
		if n.Type != xhtml.ElementNode {
			return true
		}
		switch n.DataAtom {
		case atom.Title:
			if documentTitle == "" {
				documentTitle = strings.Join(lines(n, false), " ")
			}
		case atom.Meta:
			if property := strings.ToLower(attr(n, "property")); property != "" && meta[property] == "" {
				meta[property] = strings.TrimSpace(attr(n, "content"))
			}
		case atom.H1:
			if h1 == "" {
				h1 = strings.Join(lines(n, false), " ")
			}
		}
		return true
	})
	if found := firstHeading(main); found != "" {
		h1 = found
	}
	title = extract.FirstOf(h1, meta["og:title"], documentTitle)
	company = meta["og:site_name"]
	if company == "" {
		if at := strings.LastIndex(strings.ToLower(documentTitle), " at "); at >= 0 {
			company = strings.TrimSpace(documentTitle[at+len(" at "):])
		}
	}
	return title, company
}

func firstHeading(n *xhtml.Node) string {
	found := ""
	walk(n, func(node *xhtml.Node) bool {
		if found == "" && node.Type == xhtml.ElementNode && node.DataAtom == atom.H1 {
			found = strings.Join(lines(node, false), " ")
		}
		return found == ""
	})
	return found
}

var loginPath = regexp.MustCompile(`(?i)(?:^|/)(?:login|log-in|signin|sign-in|sign_in|authwall|auth/login)(?:$|[/?.])`)

// loginPage reports a sign-in form: a password field in the main content, or
// a final address that is a login path.
func loginPage(main *xhtml.Node, final *url.URL) bool {
	if final != nil && loginPath.MatchString(final.Path) {
		return true
	}
	found := false
	walk(main, func(n *xhtml.Node) bool {
		if n.Type == xhtml.ElementNode && n.DataAtom == atom.Input && strings.EqualFold(attr(n, "type"), "password") {
			found = true
		}
		return !found
	})
	return found
}

// findPosting returns the first schema.org JobPosting in the page's JSON-LD,
// at the top level, in an array or inside @graph.
func findPosting(doc *xhtml.Node) *Page {
	var page *Page
	walk(doc, func(n *xhtml.Node) bool {
		if page != nil {
			return false
		}
		if n.Type == xhtml.ElementNode && n.DataAtom == atom.Script && strings.EqualFold(strings.TrimSpace(attr(n, "type")), "application/ld+json") {
			var data any
			if n.FirstChild != nil && json.Unmarshal([]byte(n.FirstChild.Data), &data) == nil {
				if posting := posting(data); posting != nil {
					page = fromPosting(posting)
				}
			}
			return false
		}
		return true
	})
	return page
}

func posting(data any) map[string]any {
	switch v := data.(type) {
	case []any:
		for _, item := range v {
			if found := posting(item); found != nil {
				return found
			}
		}
	case map[string]any:
		if hasType(v["@type"], "JobPosting") {
			return v
		}
		if graph, ok := v["@graph"]; ok {
			return posting(graph)
		}
	}
	return nil
}

func hasType(value any, want string) bool {
	switch v := value.(type) {
	case string:
		return v == want || strings.HasSuffix(v, "/"+want)
	case []any:
		for _, item := range v {
			if hasType(item, want) {
				return true
			}
		}
	}
	return false
}

func fromPosting(p map[string]any) *Page {
	page := &Page{ReadFrom: "json_ld", Title: text(p["title"])}
	switch org := p["hiringOrganization"].(type) {
	case map[string]any:
		page.Company = text(org["name"])
	case string:
		page.Company = text(org)
	}
	if address := firstAddress(p["jobLocation"]); address != nil {
		page.City = text(address["addressLocality"])
		switch country := address["addressCountry"].(type) {
		case map[string]any:
			page.Country = text(country["name"])
		case string:
			page.Country = text(country)
		}
	}
	if hasType(p["jobLocationType"], "TELECOMMUTE") {
		page.WorkMode = "remote"
	}
	description, _ := p["description"].(string)
	page.Text = strings.Join(descriptionLines(description), "\n")
	return page
}

func firstAddress(location any) map[string]any {
	switch v := location.(type) {
	case []any:
		for _, item := range v {
			if address := firstAddress(item); address != nil {
				return address
			}
		}
	case map[string]any:
		if address, ok := v["address"].(map[string]any); ok {
			return address
		}
	}
	return nil
}

func text(value any) string {
	s, _ := value.(string)
	return strings.TrimSpace(html.UnescapeString(s))
}

var escapedMarkup = regexp.MustCompile(`(?i)<\s*/?\s*(?:p|li|ul|ol|br|div|h[1-6]|strong|b)\b`)

// descriptionLines turns a JobPosting description (HTML, sometimes escaped
// twice) into lines; list items become "- " lines.
func descriptionLines(description string) []string {
	result := htmlLines(description)
	if joined := strings.Join(result, "\n"); escapedMarkup.MatchString(joined) {
		result = htmlLines(joined)
	}
	return result
}

func htmlLines(fragment string) []string {
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return nil
	}
	root := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	return lines(root, false)
}

// mainNode is the page's <main>, else its <article>, else its <body>.
func mainNode(doc *xhtml.Node) *xhtml.Node {
	for _, want := range []atom.Atom{atom.Main, atom.Article, atom.Body} {
		var found *xhtml.Node
		walk(doc, func(n *xhtml.Node) bool {
			if found == nil && n.Type == xhtml.ElementNode && n.DataAtom == want {
				found = n
			}
			return found == nil
		})
		if found != nil {
			return found
		}
	}
	return doc
}

var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true, atom.Svg: true,
	atom.Head: true, atom.Iframe: true,
}

var chrome = map[atom.Atom]bool{atom.Nav: true, atom.Header: true, atom.Footer: true, atom.Form: true}

var blocks = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Br: true, atom.Li: true, atom.Ul: true, atom.Ol: true, atom.Tr: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true, atom.Section: true,
	atom.Article: true, atom.Main: true, atom.Table: true, atom.Dd: true, atom.Dt: true, atom.Blockquote: true,
	atom.Pre: true, atom.Hr: true, atom.Form: true, atom.Aside: true,
}

// bold elements that start a block are heading lines of their own.
var bold = map[atom.Atom]bool{atom.Strong: true, atom.B: true}

// lines collects the text of n as lines: block elements break lines, list
// items start with "- " and a bold run that starts a block is a line of its
// own. withoutChrome also drops nav, header, footer and forms.
func lines(n *xhtml.Node, withoutChrome bool) []string {
	var out []string
	var current strings.Builder
	flush := func() {
		if line := strings.Join(strings.Fields(current.String()), " "); line != "" && line != "-" {
			out = append(out, line)
		}
		current.Reset()
	}
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		switch node.Type {
		case xhtml.TextNode:
			current.WriteString(node.Data)
			return
		case xhtml.ElementNode:
			if skipped[node.DataAtom] || (withoutChrome && chrome[node.DataAtom]) {
				return
			}
		}
		block := node.Type == xhtml.ElementNode && blocks[node.DataAtom]
		if block {
			flush()
		}
		if node.Type == xhtml.ElementNode && node.DataAtom == atom.Li {
			current.WriteString("- ")
		}
		leading := node.Type == xhtml.ElementNode && bold[node.DataAtom] && strings.TrimSpace(current.String()) == ""
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if block || leading {
			flush()
		}
	}
	visit(n)
	flush()
	return out
}

func walk(n *xhtml.Node, fn func(*xhtml.Node) bool) {
	if !fn(n) {
		return
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		walk(child, fn)
	}
}

func attr(n *xhtml.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}
