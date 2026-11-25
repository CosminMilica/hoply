package render

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"cv/internal/apperrors"
)

const maxPDF = 20 << 20

// Gotenberg prints HTML to PDF with a Gotenberg server's Chromium.
type Gotenberg struct {
	url  string
	http *http.Client
}

// NewGotenberg takes the server's base URL, such as http://gotenberg:3000.
func NewGotenberg(url string) *Gotenberg {
	return &Gotenberg{url: strings.TrimRight(url, "/"), http: &http.Client{Timeout: 30 * time.Second}}
}

// GotenbergFromEnv reads GOTENBERG_URL, defaulting to http://localhost:3000.
func GotenbergFromEnv() *Gotenberg {
	url := strings.TrimSpace(os.Getenv("GOTENBERG_URL"))
	if url == "" {
		url = "http://localhost:3000"
	}
	return NewGotenberg(url)
}

// PDF prints page on A4; the page sets its own margins.
func (g *Gotenberg) PDF(ctx context.Context, page string) ([]byte, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	if _, err := io.WriteString(file, page); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	for name, value := range map[string]string{"preferCssPageSize": "true", "printBackground": "false"} {
		if err := form.WriteField(name, value); err != nil {
			return nil, apperrors.NewInternal(err)
		}
	}
	if err := form.Close(); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url+"/forms/chromium/convert/html", &body)
	if err != nil {
		return nil, unavailable(err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	res, err := g.http.Do(req)
	if err != nil {
		return nil, unavailable(err)
	}
	defer res.Body.Close()
	pdf, err := io.ReadAll(io.LimitReader(res.Body, maxPDF))
	if err != nil {
		return nil, unavailable(err)
	}
	if res.StatusCode != http.StatusOK || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, unavailable(fmt.Errorf("answered %d", res.StatusCode))
	}
	return pdf, nil
}

func unavailable(cause error) error {
	slog.Warn("dependency failed", "dependency", "gotenberg", "error", cause)
	return apperrors.NewDependencyUnavailable("gotenberg")
}
