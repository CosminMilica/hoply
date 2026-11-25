package render

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"strings"

	"cv/internal/models"
)

// DOCX is the CV as a Word document: one section, Word's own heading styles and
// real bullets, so a recruiter can edit it and an applicant tracking system can
// read it.
// The template is one of the Templates ("" is the default).
func DOCX(cv models.CvDocument, template string) ([]byte, error) {
	look := lookOf(template).docx
	return pack(look, body(cv, look))
}

// pack zips a document body with the template's styles into a .docx.
func pack(look docxLook, documentXML string) ([]byte, error) {
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	for _, part := range []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", packageRels},
		{"word/_rels/document.xml.rels", documentRels},
		{"word/styles.xml", styles(look)},
		{"word/numbering.xml", numbering},
		{"word/document.xml", documentXML},
	} {
		file, err := archive.Create(part.name)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write([]byte(part.body)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func keepText(values ...string) []string {
	kept := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			kept = append(kept, value)
		}
	}
	return kept
}

func escape(text string) string {
	var out strings.Builder
	_ = xml.EscapeText(&out, []byte(text))
	return out.String()
}

// paragraph is one w:p with the style and the text; bold makes its run bold.
func paragraph(style, text string, bold bool) string {
	var p strings.Builder
	p.WriteString("<w:p>")
	if style != "" {
		p.WriteString(`<w:pPr><w:pStyle w:val="` + style + `"/></w:pPr>`)
	}
	p.WriteString("<w:r>")
	if bold {
		p.WriteString("<w:rPr><w:b/></w:rPr>")
	}
	p.WriteString(`<w:t xml:space="preserve">` + escape(text) + "</w:t></w:r></w:p>")
	return p.String()
}

func body(d models.CvDocument, look docxLook) string {
	var b strings.Builder
	add := func(style, text string, bold bool) {
		if strings.TrimSpace(text) != "" {
			b.WriteString(paragraph(style, text, bold))
		}
	}
	bullets := func(items []string) {
		for _, item := range items {
			add("ListBullet", item, false)
		}
	}
	if d.Name != "" {
		add("Title", d.Name, false)
		add("Contact", d.Headline, !look.centered)
	} else {
		add("Title", d.Headline, false)
	}
	add("Contact", strings.Join(d.Contact, " · "), false)
	add("Contact", strings.Join(d.Links, " · "), false)
	if strings.TrimSpace(d.Summary) != "" {
		add("Heading1", "Summary", false)
		add("", d.Summary, false)
	}
	if len(d.Skills) > 0 {
		add("Heading1", "Skills", false)
		add("", strings.Join(d.Skills, ", "), false)
	}
	if len(d.Experience) > 0 {
		add("Heading1", "Experience", false)
		for _, role := range d.Experience {
			add("Heading2", role.Role, false)
			add("", role.Meta, false)
			bullets(role.Bullets)
		}
	}
	if len(d.Projects) > 0 {
		add("Heading1", "Projects", false)
		for _, project := range d.Projects {
			add("Heading2", project.Name, false)
			add("", project.Meta, false)
			bullets(project.Bullets)
		}
	}
	if len(d.Education) > 0 {
		add("Heading1", "Education", false)
		for _, item := range d.Education {
			add("", strings.Join(keepText(item.Degree, item.Meta), " · "), false)
		}
	}
	if len(d.Certifications) > 0 {
		add("Heading1", "Certifications", false)
		bullets(d.Certifications)
	}
	if d.Languages != "" {
		add("Heading1", "Languages", false)
		add("", d.Languages, false)
	}
	if len(d.Additional) > 0 {
		add("Heading1", "Additional information", false)
		bullets(d.Additional)
	}
	return document(b.String())
}

// document is the paragraphs as word/document.xml, on A4.
func document(paragraphs string) string {
	return xml.Header + `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + paragraphs +
		`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="680" w:right="794" w:bottom="680" w:left="794" w:header="0" w:footer="0" w:gutter="0"/></w:sectPr>` +
		`</w:body></w:document>`
}

const contentTypes = xml.Header + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
	`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
	`<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>` +
	`</Types>`

const packageRels = xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`</Relationships>`

const documentRels = xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
	`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>` +
	`</Relationships>`

func styles(look docxLook) string {
	center, caps, titleCaps := "", "", ""
	if look.centered {
		center = `<w:jc w:val="center"/>`
	}
	if look.headingCaps {
		caps = "<w:caps/>"
	}
	if look.titleCaps {
		titleCaps = `<w:caps/><w:spacing w:val="40"/>`
	}
	titleWeight := "<w:b/>"
	if look.titleCaps {
		titleWeight = ""
	}
	return xml.Header + `<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="` + look.font + `" w:hAnsi="` + look.font + `" w:cs="` + look.font + `"/><w:sz w:val="20"/><w:szCs w:val="20"/></w:rPr></w:rPrDefault>` +
		`<w:pPrDefault><w:pPr><w:spacing w:after="40" w:line="264" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
		`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:pPr>` + center + `</w:pPr><w:rPr>` + titleWeight + titleCaps +
		`<w:color w:val="` + look.accent + `"/><w:sz w:val="36"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Contact"><w:name w:val="Contact"/><w:basedOn w:val="Normal"/><w:pPr>` + center + `</w:pPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="200" w:after="60"/>` +
		`<w:pBdr>` + look.headingRules + `</w:pBdr>` + center + `<w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/>` + caps + `<w:color w:val="` + look.accent + `"/><w:sz w:val="19"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="120" w:after="0"/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:b/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Date"><w:name w:val="Date"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="280" w:after="240"/></w:pPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Body"><w:name w:val="Body Text"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="180" w:line="300" w:lineRule="auto"/></w:pPr><w:rPr><w:sz w:val="21"/><w:szCs w:val="21"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="ListBullet"><w:name w:val="List Bullet"/><w:basedOn w:val="Normal"/><w:pPr><w:numPr><w:numId w:val="1"/></w:numPr><w:ind w:left="360" w:hanging="240"/></w:pPr></w:style>` +
		`</w:styles>`
}

const numbering = xml.Header + `<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:abstractNum w:abstractNumId="0"><w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="•"/><w:lvlJc w:val="left"/>` +
	`<w:pPr><w:ind w:left="360" w:hanging="240"/></w:pPr></w:lvl></w:abstractNum>` +
	`<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>` +
	`</w:numbering>`
