package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"site-analyzer/internal/model"
)

func TestRunValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
	}{
		{name: "help", args: []string{"-h"}, wantCode: 0, wantOutput: "Usage:"},
		{name: "no targets", args: nil, wantCode: 2, wantOutput: "no targets"},
		{name: "bad concurrency", args: []string{"-c", "0", "example.com"}, wantCode: 2, wantOutput: "concurrency"},
		{name: "bad format", args: []string{"-f", "xml", "example.com"}, wantCode: 2, wantOutput: "unsupported report format"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), test.args, strings.NewReader(""), &stdout, &stderr)
			combined := stdout.String() + stderr.String()
			if code != test.wantCode {
				t.Fatalf("Run() code = %d, want %d; output: %s", code, test.wantCode, combined)
			}
			if !strings.Contains(combined, test.wantOutput) {
				t.Fatalf("Run() output %q does not contain %q", combined, test.wantOutput)
			}
		})
	}
}

func TestRunEndToEnd(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Scan-Test"); got != "yes" {
			t.Errorf("X-Scan-Test = %q, want yes", got)
		}
		w.Header().Set("Server", "nginx")
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte("<html><head><title>CLI test</title></head></html>")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := Run(
		context.Background(),
		[]string{"-u", server.URL, "-H", "X-Scan-Test: yes", "-f", "jsonl", "--quiet"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr: %s", code, stderr.String())
	}
	var result model.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if result.Title != "CLI test" || result.StatusCode != http.StatusOK {
		t.Fatalf("result = %+v, want title CLI test and status 200", result)
	}
}

func TestRunWritesOutputAndReportsTargetFailure(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "report.json")
	var stdout, stderr bytes.Buffer
	code := Run(
		context.Background(),
		[]string{"-u", "ftp://example.com", "-f", "json", "-o", path, "--quiet"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1; stderr: %s", code, stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var results []model.Result
	if err := json.Unmarshal(data, &results); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if len(results) != 1 || !strings.Contains(results[0].Error, "unsupported URL scheme") {
		t.Fatalf("results = %+v, want one unsupported scheme error", results)
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--version"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "site-analyzer") {
		t.Fatalf("Run() code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}
