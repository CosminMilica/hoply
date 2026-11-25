package unit_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cv/internal/apperrors"
	"cv/internal/cvdoc"
	"cv/internal/models"
	"cv/internal/render"
)

func sampleDocument(t *testing.T) models.CvDocument {
	t.Helper()
	profile := richProfile(t)
	profile.About = "Builds <script>alert(1)</script> & ships."
	return cvdoc.Write(profile, engineJob(t, 0), request(), now).Profile
}

func TestHTMLHasEverySectionInOneColumnAndEscapesText(t *testing.T) {
	page, err := render.HTML(sampleDocument(t), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<title>Andrei Popescu – CV</title>", `<p class="name">Andrei Popescu</p>`, "<h2>Summary</h2>", "<h2>Skills</h2>",
		"<h2>Experience</h2>", "<h3>DevOps Engineer</h3>", "<h2>Projects</h2>", "<h2>Education</h2>", "<h2>Certifications</h2>",
		"<h2>Languages</h2>", "<h2>Additional information</h2>", "&lt;script&gt;alert(1)&lt;/script&gt; &amp; ships.", "andrei@example.com · &#43;40 700 000 000 · Iași, Romania"} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(page, "<script>") || strings.Contains(page, "<table") || strings.Contains(page, "<img") {
		t.Fatal("no script, table or image")
	}
	thin, err := render.HTML(models.CvDocument{Headline: "Engineer"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(thin, `<p class="name">Engineer</p>`) || !strings.Contains(thin, "<title>CV</title>") {
		t.Fatalf("a thin CV: %s", thin)
	}
	for _, empty := range []string{"Summary", "Skills", "Experience", "Projects", "Education", "Certifications", "Languages", "Additional information"} {
		if strings.Contains(thin, "<h2>"+empty+"</h2>") {
			t.Errorf("a section without data is left out: %s", empty)
		}
	}
}

func TestDOCXIsAWordDocumentWithTheCVText(t *testing.T) {
	raw, err := render.DOCX(sampleDocument(t), "")
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(reader)
		reader.Close()
		parts[file.Name] = string(body)
		decoder := xml.NewDecoder(bytes.NewReader(body))
		for {
			if _, err := decoder.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s is not well-formed XML: %v", file.Name, err)
			}
		}
	}
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/_rels/document.xml.rels", "word/styles.xml", "word/numbering.xml", "word/document.xml"} {
		if _, ok := parts[name]; !ok {
			t.Fatalf("missing part %s", name)
		}
	}
	document := parts["word/document.xml"]
	for _, want := range []string{`<w:pStyle w:val="Title"/></w:pPr><w:r><w:t xml:space="preserve">Andrei Popescu</w:t>`, ">Summary<", ">Experience<",
		`<w:pStyle w:val="ListBullet"/>`, "&lt;script&gt;alert(1)&lt;/script&gt; &amp; ships.", ">Additional information<"} {
		if !strings.Contains(document, want) {
			t.Errorf("document lacks %q", want)
		}
	}
	thin, err := render.DOCX(models.CvDocument{Headline: "Engineer"}, "")
	if err != nil || !bytes.HasPrefix(thin, []byte("PK")) {
		t.Fatalf("a thin CV: %v", err)
	}
	if text := docxPart(t, thin, "word/document.xml"); strings.Contains(text, ">Summary<") {
		t.Fatal("an empty summary is left out")
	}
}

func TestGotenbergPrintsThePage(t *testing.T) {
	var got struct{ path, page, preferCSS string }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
		}
		file, _, err := r.FormFile("files")
		if err == nil {
			body, _ := io.ReadAll(file)
			got.page = string(body)
		}
		got.preferCSS = r.FormValue("preferCssPageSize")
		_, _ = w.Write([]byte("%PDF-1.4 fake"))
	}))
	defer server.Close()
	pdf, err := render.NewGotenberg(server.URL+"/").PDF(context.Background(), "<p>Hi</p>")
	if err != nil || string(pdf) != "%PDF-1.4 fake" || got.path != "/forms/chromium/convert/html" || got.page != "<p>Hi</p>" || got.preferCSS != "true" {
		t.Fatalf("pdf: %q %v %+v", pdf, err, got)
	}
}

func TestGotenbergFailuresAreUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	var appErr *apperrors.Error
	if _, err := render.NewGotenberg(server.URL).PDF(context.Background(), "<p/>"); !errors.As(err, &appErr) || appErr.Code != apperrors.DependencyUnavailable {
		t.Fatalf("a failing server: %v", err)
	}
	notPDF := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }))
	defer notPDF.Close()
	if _, err := render.NewGotenberg(notPDF.URL).PDF(context.Background(), "<p/>"); !errors.As(err, &appErr) {
		t.Fatalf("not a PDF: %v", err)
	}
	server.Close()
	if _, err := render.NewGotenberg(server.URL).PDF(context.Background(), "<p/>"); !errors.As(err, &appErr) {
		t.Fatalf("unreachable: %v", err)
	}
	if _, err := render.NewGotenberg("http://bad host").PDF(context.Background(), "<p/>"); !errors.As(err, &appErr) {
		t.Fatalf("bad URL: %v", err)
	}
}

func TestGotenbergFromEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("%PDF-x")) }))
	defer server.Close()
	t.Setenv("GOTENBERG_URL", server.URL)
	if pdf, err := render.GotenbergFromEnv().PDF(context.Background(), "<p/>"); err != nil || string(pdf) != "%PDF-x" {
		t.Fatalf("env: %q %v", pdf, err)
	}
	t.Setenv("GOTENBERG_URL", "")
	if render.GotenbergFromEnv() == nil {
		t.Fatal("a default URL")
	}
}

func docxPart(t *testing.T, raw []byte, name string) string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name == name {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			body, _ := io.ReadAll(reader)
			return string(body)
		}
	}
	t.Fatalf("missing part %s", name)
	return ""
}

func TestEveryTemplateKeepsTheSameTextAndLooksItsOwn(t *testing.T) {
	document := sampleDocument(t)
	classic, _ := render.HTML(document, "")
	for _, tc := range []struct{ template, css, font string }{
		{"classic", "", "Arial"}, {"modern", "Carlito", "Calibri"}, {"elegant", "Caladea", "Cambria"},
	} {
		page, err := render.HTML(document, tc.template)
		if err != nil {
			t.Fatal(err)
		}
		if tc.css != "" && !strings.Contains(page, tc.css) {
			t.Errorf("%s page lacks %s", tc.template, tc.css)
		}
		if body := page[strings.Index(page, "<body>"):]; body != classic[strings.Index(classic, "<body>"):] {
			t.Errorf("%s changes the text, not only the look", tc.template)
		}
		raw, err := render.DOCX(document, tc.template)
		if err != nil {
			t.Fatal(err)
		}
		if styles := docxPart(t, raw, "word/styles.xml"); !strings.Contains(styles, `w:ascii="`+tc.font+`"`) {
			t.Errorf("%s DOCX is not in %s", tc.template, tc.font)
		}
	}
	elegant, _ := render.DOCX(document, "elegant")
	if !strings.Contains(docxPart(t, elegant, "word/styles.xml"), `<w:jc w:val="center"/>`) {
		t.Error("the elegant header is centered")
	}
	equal(t, render.Known(""), true)
	equal(t, render.Known("modern"), true)
	equal(t, render.Known("fancy"), false)
	equal(t, render.Templates, []string{"classic", "modern", "elegant"})
}
