// Package render turns a CV document into the files a person sends: a PDF
// (HTML printed by Gotenberg's Chromium) and a DOCX. Both are one column with
// standard headings and real text, as applicant tracking systems read them, and
// both follow the web app's CV paper (apps/frontend/src/features/cv/components/CvPaper.vue).
package render

import (
	"bytes"
	htmltemplate "html/template"
	"strings"

	"cv/internal/cvdoc"
	"cv/internal/models"
)

// baseCSS is the classic look; a template's CSS comes after it.
const baseCSS = `
  @page { size: A4; margin: 12mm 14mm; }
  * { box-sizing: border-box; }
  body { margin: 0; font-family: "Liberation Sans", Arial, Helvetica, sans-serif; font-size: 10pt; line-height: 1.32; color: #111111; }
  header { margin-bottom: 6pt; }
  .name { font-size: 17pt; font-weight: 700; margin: 0; }
  .headline { font-size: 10.5pt; font-weight: 700; margin: 0; }
  .line { margin: 0; }
  h2 { margin: 8pt 0 3pt; padding-bottom: 1.5pt; border-bottom: 1px solid #111111; font-size: 9pt; letter-spacing: 0.08em; text-transform: uppercase; }
  h3 { margin: 4pt 0 0; font-size: 10pt; }
  p { margin: 0; }
  ul { margin: 1pt 0 0; padding-left: 13pt; }
  li { margin: 0.5pt 0; }
  .meta { font-weight: 400; }
  .entry { break-inside: avoid; }
`

var page = htmltemplate.Must(htmltemplate.New("cv").Funcs(htmltemplate.FuncMap{"join": strings.Join}).Parse(`<!doctype html>
<html lang="{{.Language}}">
<head>
<meta charset="utf-8">
<title>{{if .Name}}{{.Name}} – CV{{else}}CV{{end}}</title>
<style>
{{.CSS}}
</style>
</head>
<body>
<header>
{{- if .Name}}
  <p class="name">{{.Name}}</p>
  <p class="headline">{{.Headline}}</p>
{{- else}}
  <p class="name">{{.Headline}}</p>
{{- end}}
{{- if .Contact}}
  <p class="line">{{join .Contact " · "}}</p>
{{- end}}
{{- if .Links}}
  <p class="line">{{join .Links " · "}}</p>
{{- end}}
</header>
{{- if .Summary}}
<section>
  <h2>{{.Headings.Summary}}</h2>
  <p>{{.Summary}}</p>
</section>
{{- end}}
{{- if .Skills}}
<section>
  <h2>{{.Headings.Skills}}</h2>
  <p>{{join .Skills ", "}}</p>
</section>
{{- end}}
{{- if .Experience}}
<section>
  <h2>{{.Headings.Experience}}</h2>
{{- range .Experience}}
  <div class="entry">
    <h3>{{.Role}}</h3>
    <p>{{.Meta}}</p>
    <ul>{{range .Bullets}}<li>{{.}}</li>{{end}}</ul>
  </div>
{{- end}}
</section>
{{- end}}
{{- if .Projects}}
<section>
  <h2>{{.Headings.Projects}}</h2>
{{- range .Projects}}
  <div class="entry">
    <h3>{{.Name}}</h3>
{{- if .Meta}}
    <p>{{.Meta}}</p>
{{- end}}
    <ul>{{range .Bullets}}<li>{{.}}</li>{{end}}</ul>
  </div>
{{- end}}
</section>
{{- end}}
{{- if .Education}}
<section>
  <h2>{{.Headings.Education}}</h2>
{{- range .Education}}
  <p><strong>{{.Degree}}</strong>{{if .Meta}} · {{.Meta}}{{end}}</p>
{{- end}}
</section>
{{- end}}
{{- if .Certifications}}
<section>
  <h2>{{.Headings.Certifications}}</h2>
  <ul>{{range .Certifications}}<li>{{.}}</li>{{end}}</ul>
</section>
{{- end}}
{{- if .Languages}}
<section>
  <h2>{{.Headings.Languages}}</h2>
  <p>{{.Languages}}</p>
</section>
{{- end}}
{{- if .Additional}}
<section>
  <h2>{{.Headings.Additional}}</h2>
  <ul>{{range .Additional}}<li>{{.}}</li>{{end}}</ul>
</section>
{{- end}}
</body>
</html>
`))

// HTML is the CV as a printable page in one of the Templates ("" is the default).
func HTML(document models.CvDocument, template string) (string, error) {
	var out bytes.Buffer
	document.Language = cvdoc.LanguageOf(document.Language)
	if err := page.Execute(&out, struct {
		models.CvDocument
		Headings cvdoc.Headings
		CSS      htmltemplate.CSS
	}{document, cvdoc.HeadingsFor(document.Language), htmltemplate.CSS(baseCSS) + lookOf(template).css}); err != nil {
		return "", err
	}
	return out.String(), nil
}
