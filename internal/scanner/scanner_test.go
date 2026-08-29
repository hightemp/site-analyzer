package scanner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "adds https", input: "example.com/path", want: "https://example.com/path"},
		{name: "keeps http", input: "http://example.com", want: "http://example.com"},
		{name: "drops fragment", input: "https://example.com/#part", want: "https://example.com/"},
		{name: "rejects ftp", input: "ftp://example.com", wantErr: true},
		{name: "rejects empty", input: " ", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeURL(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("NormalizeURL() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("NormalizeURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestScannerScan(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "nginx")
		w.Header().Set("X-Powered-By", "PHP/8.2")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := w.Write([]byte(`<!doctype html><html><head><title> Test  page </title><meta name="generator" content="WordPress 6.5"></head></html>`))
		if err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	s, err := New(Options{
		Timeout:         2 * time.Second,
		MaxBodyBytes:    1024 * 1024,
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := s.Scan(context.Background(), server.URL)
	if result.Error != "" {
		t.Fatalf("Scan() error = %s", result.Error)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Scan() status = %d, want %d", result.StatusCode, http.StatusOK)
	}
	if result.Title != "Test page" {
		t.Fatalf("Scan() title = %q, want %q", result.Title, "Test page")
	}
	if len(result.Technologies) == 0 {
		t.Fatal("Scan() detected no technologies")
	}

	var names []string
	for _, technology := range result.Technologies {
		names = append(names, technology.Name)
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "PHP") && !strings.Contains(joined, "WordPress") && !strings.Contains(joined, "Nginx") {
		t.Fatalf("Scan() technologies = %q, expected PHP, WordPress, or Nginx", joined)
	}
}

func TestNewValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options Options
	}{
		{name: "zero timeout", options: Options{MaxBodyBytes: 1}},
		{name: "zero body limit", options: Options{Timeout: time.Second}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := New(test.options); err == nil {
				t.Fatal("New() error = nil, want validation error")
			}
		})
	}
}

func TestScannerOptionsAndTruncation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		if r.Header.Get("User-Agent") != "test-agent" || r.Header.Get("X-Test") != "present" {
			t.Errorf("unexpected headers: %v", r.Header)
		}
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(strings.Repeat("x", 100))); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	s, err := New(Options{
		Timeout:         2 * time.Second,
		MaxBodyBytes:    10,
		UserAgent:       "test-agent",
		FollowRedirects: false,
		Headers:         http.Header{"X-Test": []string{"present"}},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := s.Scan(context.Background(), server.URL)
	if result.Error != "" || !result.Truncated || result.BodyBytes != 10 {
		t.Fatalf("Scan() = %+v, want truncated 10-byte result", result)
	}

	redirect := s.Scan(context.Background(), server.URL+"/redirect")
	if redirect.Error != "" || redirect.StatusCode != http.StatusFound {
		t.Fatalf("redirect Scan() = %+v, want status 302", redirect)
	}
}

func TestScannerScanInvalidTarget(t *testing.T) {
	t.Parallel()

	s, err := New(Options{Timeout: time.Second, MaxBodyBytes: 1024})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result := s.Scan(context.Background(), "ftp://example.com")
	if !strings.Contains(result.Error, "unsupported URL scheme") || result.DurationMS < 0 {
		t.Fatalf("Scan() = %+v, want unsupported scheme error", result)
	}
}
