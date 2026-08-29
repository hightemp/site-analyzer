// Package scanner fetches websites and detects their technologies.
package scanner

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
	"golang.org/x/net/html"

	"site-analyzer/internal/model"
)

const defaultUserAgent = "site-analyzer/1.0"

// Options configures HTTP fetching and response processing.
type Options struct {
	Timeout         time.Duration
	MaxBodyBytes    int64
	UserAgent       string
	Insecure        bool
	FollowRedirects bool
	Headers         http.Header
}

// Scanner fetches targets and identifies their technology stack.
type Scanner struct {
	httpClient *http.Client
	detector   *wappalyzer.Wappalyze
	options    Options
}

// New constructs a Scanner and loads the embedded Wappalyzer fingerprints.
func New(options Options) (*Scanner, error) {
	if options.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}
	if options.MaxBodyBytes <= 0 {
		return nil, fmt.Errorf("maximum body size must be greater than zero")
	}
	if options.UserAgent == "" {
		options.UserAgent = defaultUserAgent
	}

	detector, err := wappalyzer.New()
	if err != nil {
		return nil, fmt.Errorf("initialize wappalyzer: %w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if options.Insecure {
		// InsecureSkipVerify is intentionally exposed as an explicit CLI option
		// for scanning development systems with self-signed certificates.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	client := &http.Client{Transport: transport, Timeout: options.Timeout}
	if !options.FollowRedirects {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &Scanner{httpClient: client, detector: detector, options: options}, nil
}

// Scan fetches one target. Fetch and validation failures are stored in Result
// so a bad target does not stop a batch scan.
func (s *Scanner) Scan(ctx context.Context, input string) (result model.Result) {
	started := time.Now()
	result = model.Result{
		InputURL:     input,
		Technologies: make([]model.Technology, 0),
		ScannedAt:    started.UTC(),
	}
	defer func() {
		result.DurationMS = time.Since(started).Milliseconds()
	}()

	normalized, err := NormalizeURL(input)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.URL = normalized

	requestCtx, cancel := context.WithTimeout(ctx, s.options.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, normalized, nil)
	if err != nil {
		result.Error = fmt.Sprintf("create request: %v", err)
		return result
	}
	for name, values := range s.options.Headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", s.options.UserAgent)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}

	result.FinalURL = resp.Request.URL.String()
	result.StatusCode = resp.StatusCode
	result.Server = resp.Header.Get("Server")
	result.ContentType = resp.Header.Get("Content-Type")

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, s.options.MaxBodyBytes+1))
	closeErr := resp.Body.Close()
	if readErr != nil {
		result.Error = fmt.Sprintf("read response body: %v", readErr)
		return result
	}
	if closeErr != nil {
		result.Error = fmt.Sprintf("close response body: %v", closeErr)
		return result
	}
	if int64(len(body)) > s.options.MaxBodyBytes {
		body = body[:s.options.MaxBodyBytes]
		result.Truncated = true
	}
	result.BodyBytes = int64(len(body))
	result.Title = documentTitle(body)
	result.Technologies = technologyList(s.detector.FingerprintWithInfo(resp.Header, body))
	return result
}

// NormalizeURL adds https:// when the scheme is omitted and validates that the
// target is an HTTP(S) URL with a host.
func NormalizeURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("target is empty")
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid target %q: %w", value, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}
	if parsed.Hostname() == "" {
		return "", fmt.Errorf("target %q has no host", value)
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func technologyList(apps map[string]wappalyzer.AppInfo) []model.Technology {
	technologies := make([]model.Technology, 0, len(apps))
	for app, info := range apps {
		name, version, _ := strings.Cut(app, ":")
		categories := append([]string(nil), info.Categories...)
		sort.Strings(categories)
		technologies = append(technologies, model.Technology{
			Name:        name,
			Version:     version,
			Categories:  categories,
			Website:     info.Website,
			Description: info.Description,
			CPE:         info.CPE,
		})
	}
	sort.Slice(technologies, func(i, j int) bool {
		if technologies[i].Name == technologies[j].Name {
			return technologies[i].Version < technologies[j].Version
		}
		return technologies[i].Name < technologies[j].Name
	})
	return technologies
}

func documentTitle(body []byte) string {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var visit func(*html.Node) string
	visit = func(node *html.Node) string {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "title") {
			if node.FirstChild != nil {
				return strings.Join(strings.Fields(node.FirstChild.Data), " ")
			}
			return ""
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if title := visit(child); title != "" {
				return title
			}
		}
		return ""
	}
	return visit(doc)
}
