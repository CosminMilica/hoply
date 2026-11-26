// Package fetch reads a job page from a link: it refuses addresses inside the
// network on every connection (redirects included), caps time, size and
// redirects, and turns the page into text lines from its schema.org
// JobPosting or, failing that, its main text.
package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"syscall"
	"time"
)

// Reason says why a page could not be read.
type Reason string

const (
	LoginRequired  Reason = "login_required"
	NoJobData      Reason = "no_job_data"
	FetchFailed    Reason = "fetch_failed"
	TooLarge       Reason = "too_large"
	BlockedAddress Reason = "blocked_address"
)

// Error is a page that could not be read.
type Error struct {
	Reason Reason
	Err    error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return "page unreadable: " + string(e.Reason)
	}
	return fmt.Sprintf("page unreadable: %s: %v", e.Reason, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func fail(reason Reason, err error) *Error { return &Error{Reason: reason, Err: err} }

// Page is a job page turned into text. ReadFrom is "ats_api", "json_ld" or
// "page_text"; the other fields are what the board API, the JobPosting or the
// page's title tags say, and may be empty.
type Page struct {
	ReadFrom string
	Text     string
	Title    string
	Company  string
	Location string
	City     string
	Country  string
	WorkMode string
}

// Options tune a Reader. Zero values take the defaults.
type Options struct {
	Timeout  time.Duration
	MaxBytes int64
	// MaxAPIBytes caps a job board API answer, which may hold a whole board.
	MaxAPIBytes  int64
	MaxRedirects int
	UserAgent    string
	// AllowLoopback lets tests reach an httptest server on 127.0.0.1 or ::1.
	// Production code never sets it.
	AllowLoopback bool
	// Hosts sends requests for a host (the key) to another base URL instead,
	// so tests can stand in for the job board APIs. Production code never sets
	// it; redirects are not rewritten.
	Hosts map[string]string
}

const (
	defaultTimeout      = 10 * time.Second
	defaultMaxBytes     = 2 << 20
	defaultMaxAPIBytes  = 10 << 20
	defaultMaxRedirects = 3
	defaultUserAgent    = "Hoply/0.1 (job reader)"
	minMainText         = 300
)

// Reader fetches and reads job pages.
type Reader struct {
	client *http.Client
	opts   Options
}

func New(opts Options) *Reader {
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = defaultMaxBytes
	}
	if opts.MaxAPIBytes <= 0 {
		opts.MaxAPIBytes = defaultMaxAPIBytes
	}
	if opts.MaxRedirects <= 0 {
		opts.MaxRedirects = defaultMaxRedirects
	}
	if opts.UserAgent == "" {
		opts.UserAgent = defaultUserAgent
	}
	dialer := &net.Dialer{Timeout: opts.Timeout, Control: control(opts.AllowLoopback)}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   opts.Timeout,
		ResponseHeaderTimeout: opts.Timeout,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	r := &Reader{opts: opts}
	r.client = &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > opts.MaxRedirects {
				return fail(FetchFailed, fmt.Errorf("more than %d redirects", opts.MaxRedirects))
			}
			return checkURL(req.URL)
		},
	}
	return r
}

// control refuses a connection to an address inside the network. It runs
// after the host name is resolved, for every dial, so a redirect or a DNS
// answer cannot reach a refused address.
func control(allowLoopback bool) func(string, string, syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fail(BlockedAddress, err)
		}
		ip, err := netip.ParseAddr(host)
		if err != nil {
			return fail(BlockedAddress, err)
		}
		ip = ip.Unmap()
		if allowLoopback && ip.IsLoopback() {
			return nil
		}
		if Refused(ip) {
			return fail(BlockedAddress, fmt.Errorf("address %s is not public", ip))
		}
		return nil
	}
}

var refusedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2002::/16"),
}

// Refused reports whether ip is loopback, private, link-local, multicast,
// unspecified or another range that does not reach the public internet.
func Refused(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, prefix := range refusedPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return ip == netip.MustParseAddr("255.255.255.255")
}

// LoginWalled reports hosts that show job pages only to signed-in members.
func LoginWalled(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "linkedin.com" || strings.HasSuffix(host, ".linkedin.com") {
		return true
	}
	for _, label := range strings.Split(host, ".") {
		if label == "glassdoor" {
			return true
		}
	}
	return false
}

func checkURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fail(FetchFailed, fmt.Errorf("scheme %q is not http(s)", u.Scheme))
	}
	if u.Hostname() == "" {
		return fail(FetchFailed, errors.New("the link has no host"))
	}
	if LoginWalled(u.Hostname()) {
		return fail(LoginRequired, nil)
	}
	return nil
}

// Read fetches rawURL and reads the job in it: through the job board's API
// when the link is a known board's, else from the page.
func (r *Reader) Read(ctx context.Context, rawURL string) (*Page, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, fail(FetchFailed, err)
	}
	if err := checkURL(target); err != nil {
		return nil, err
	}
	if board := boardFor(target); board != nil {
		if page, err := r.readBoard(ctx, board); err == nil {
			return page, nil
		}
	}
	body, final, err := r.get(ctx, target, "text/html", r.opts.MaxBytes)
	if err != nil {
		return nil, err
	}
	return readPage(body, final)
}

func (r *Reader) readBoard(ctx context.Context, board *board) (*Page, error) {
	api, err := url.Parse(board.api)
	if err != nil {
		return nil, fail(FetchFailed, err)
	}
	body, _, err := r.get(ctx, api, "application/json", r.opts.MaxAPIBytes)
	if err != nil {
		return nil, err
	}
	page, err := board.read(body)
	if err != nil {
		return nil, fail(NoJobData, err)
	}
	page.ReadFrom = "ats_api"
	return page, nil
}

// get fetches target and returns its body when the response is a 2xx of the
// media type wanted, of at most limit bytes.
func (r *Reader) get(ctx context.Context, target *url.URL, media string, limit int64) ([]byte, *url.URL, error) {
	return r.send(ctx, http.MethodGet, target, nil, limit, media)
}

// send makes a request, with a JSON body when body is not nil, and returns the
// response body when it is a 2xx of one of the media types, of at most limit
// bytes.
func (r *Reader) send(ctx context.Context, method string, target *url.URL, body []byte, limit int64, media ...string) ([]byte, *url.URL, error) {
	address := target.String()
	if base, ok := r.opts.Hosts[target.Host]; ok {
		rewritten := *target
		if b, err := url.Parse(base); err == nil {
			rewritten.Scheme, rewritten.Host = b.Scheme, b.Host
		}
		address = rewritten.String()
	}
	var payload io.Reader
	if body != nil {
		payload = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, address, payload)
	if err != nil {
		return nil, nil, fail(FetchFailed, err)
	}
	req.Header.Set("User-Agent", r.opts.UserAgent)
	req.Header.Set("Accept", strings.Join(media, ", "))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		var unreadable *Error
		if errors.As(err, &unreadable) {
			return nil, nil, unreadable
		}
		return nil, nil, fail(FetchFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil, fail(FetchFailed, fmt.Errorf("status %d", resp.StatusCode))
	}
	if got, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err != nil || !slices.Contains(media, got) {
		return nil, nil, fail(FetchFailed, fmt.Errorf("content type %q is not %s", resp.Header.Get("Content-Type"), strings.Join(media, " or ")))
	}
	if resp.ContentLength > limit {
		return nil, nil, fail(TooLarge, nil)
	}
	var answer bytes.Buffer
	if _, err := io.Copy(&answer, io.LimitReader(resp.Body, limit+1)); err != nil {
		return nil, nil, fail(FetchFailed, err)
	}
	if int64(answer.Len()) > limit {
		return nil, nil, fail(TooLarge, nil)
	}
	return answer.Bytes(), resp.Request.URL, nil
}
