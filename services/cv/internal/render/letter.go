package render

import (
	"bytes"
	htmltemplate "html/template"
	"regexp"
	"strings"

	"cv/internal/models"
)

// Letter is a cover letter to export: the CV's header, the date and the text
// as the person edited it, paragraphs separated by a blank line.
type Letter struct {
	Header models.CvDocument
	Date   string
	Text   string
}

var blankLine = regexp.MustCompile(`\n[ \t]*\n`)

// paragraphs are the letter's paragraphs, each as its lines.
func (l Letter) paragraphs() [][]string {
	paragraphs := [][]string{}
	for _, block := range blankLine.Split(strings.ReplaceAll(strings.TrimSpace(l.Text), "\r\n", "\n"), -1) {
		lines := []string{}
		for _, line := range strings.Split(block, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			paragraphs = append(paragraphs, lines)
		}
	}
	return paragraphs
}

const letterCSS = `
  header { margin-bottom: 16pt; }
  .date { margin: 0 0 12pt; }
  .para { margin: 0 0 9pt; font-size: 10.5pt; line-height: 1.45; }
`

var letterPage = htmltemplate.Must(htmltemplate.New("letter").Funcs(htmltemplate.FuncMap{"join": strings.Join}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{if .Header.Name}}{{.Header.Name}} – Cover letter{{else}}Cover letter{{end}}</title>
<style>
{{.CSS}}
</style>
</head>
<body>
<header>
{{- if .Header.Name}}
  <p class="name">{{.Header.Name}}</p>
  <p class="headline">{{.Header.Headline}}</p>
{{- else}}
  <p class="name">{{.Header.Headline}}</p>
{{- end}}
{{- if .Header.Contact}}
  <p class="line">{{join .Header.Contact " · "}}</p>
{{- end}}
{{- if .Header.Links}}
  <p class="line">{{join .Header.Links " · "}}</p>
{{- end}}
</header>
<p class="date">{{.Date}}</p>
{{- range .Paragraphs}}
<p class="para">{{range $i, $line := .}}{{if $i}}<br>{{end}}{{$line}}{{end}}</p>
{{- end}}
</body>
</html>
`))

// LetterHTML is the cover letter as a printable page in one of the Templates.
func LetterHTML(letter Letter, template string) (string, error) {
	var out bytes.Buffer
	err := letterPage.Execute(&out, struct {
		Header     models.CvDocument
		Date       string
		Paragraphs [][]string
		CSS        htmltemplate.CSS
	}{letter.Header, letter.Date, letter.paragraphs(), htmltemplate.CSS(baseCSS) + lookOf(template).css + letterCSS})
	return out.String(), err
}

// LetterDOCX is the cover letter as a Word document in one of the Templates.
func LetterDOCX(letter Letter, template string) ([]byte, error) {
	look := lookOf(template).docx
	var b strings.Builder
	add := func(style, text string, bold bool) {
		if strings.TrimSpace(text) != "" {
			b.WriteString(paragraph(style, text, bold))
		}
	}
	h := letter.Header
	if h.Name != "" {
		add("Title", h.Name, false)
		add("Contact", h.Headline, !look.centered)
	} else {
		add("Title", h.Headline, false)
	}
	add("Contact", strings.Join(h.Contact, " · "), false)
	add("Contact", strings.Join(h.Links, " · "), false)
	add("Date", letter.Date, false)
	for _, lines := range letter.paragraphs() {
		b.WriteString(`<w:p><w:pPr><w:pStyle w:val="Body"/></w:pPr>`)
		for i, line := range lines {
			if i > 0 {
				b.WriteString("<w:r><w:br/></w:r>")
			}
			b.WriteString(`<w:r><w:t xml:space="preserve">` + escape(line) + "</w:t></w:r>")
		}
		b.WriteString("</w:p>")
	}
	return pack(look, document(b.String()))
}
