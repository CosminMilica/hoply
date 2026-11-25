package render

import "html/template"

// Templates are the looks a CV can be exported in, the default first. Every one
// keeps the same single column, standard headings and real text; only type,
// color and alignment change.
var Templates = []string{"classic", "modern", "elegant"}

// look is one template: the page's CSS on top of the base, and the Word styles.
type look struct {
	css  template.CSS
	docx docxLook
}

type docxLook struct {
	font         string
	accent       string
	centered     bool
	titleCaps    bool
	headingCaps  bool
	headingRules string
}

var looks = map[string]look{
	"classic": {
		docx: docxLook{font: "Arial", accent: "111111", headingCaps: true,
			headingRules: `<w:bottom w:val="single" w:sz="6" w:space="1" w:color="111111"/>`},
	},
	"modern": {
		css: `
  body { font-family: Carlito, Calibri, "Liberation Sans", Arial, sans-serif; font-size: 10.5pt; color: #1f2933; }
  header { padding-bottom: 5pt; border-bottom: 2pt solid #1d4e89; }
  .name { font-size: 21pt; color: #1d4e89; }
  .headline { font-size: 11.5pt; font-weight: 400; color: #1f2933; }
  h2 { margin-top: 9pt; padding: 0 0 0 5pt; border-bottom: none; border-left: 3pt solid #1d4e89; color: #1d4e89; font-size: 11pt; letter-spacing: 0; text-transform: none; }
  h3 { color: #111111; }`,
		docx: docxLook{font: "Calibri", accent: "1D4E89",
			headingRules: `<w:left w:val="single" w:sz="24" w:space="4" w:color="1D4E89"/>`},
	},
	"elegant": {
		css: `
  body { font-family: Caladea, Cambria, "Liberation Serif", Georgia, serif; font-size: 10.5pt; }
  header { text-align: center; }
  .name { font-size: 19pt; font-weight: 400; letter-spacing: 0.14em; text-transform: uppercase; }
  .headline { font-style: italic; font-weight: 400; }
  h2 { padding: 1.5pt 0; border-top: 0.75pt solid #444444; border-bottom: 0.75pt solid #444444; font-weight: 400; letter-spacing: 0.2em; text-align: center; }`,
		docx: docxLook{font: "Cambria", accent: "111111", centered: true, titleCaps: true, headingCaps: true,
			headingRules: `<w:top w:val="single" w:sz="4" w:space="1" w:color="444444"/><w:bottom w:val="single" w:sz="4" w:space="1" w:color="444444"/>`},
	},
}

// Known reports whether name is a template; "" is the default.
func Known(name string) bool {
	_, ok := looks[orDefault(name)]
	return ok
}

func orDefault(name string) string {
	if name == "" {
		return Templates[0]
	}
	return name
}

func lookOf(name string) look {
	if found, ok := looks[orDefault(name)]; ok {
		return found
	}
	return looks[Templates[0]]
}
