package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"site-analyzer/internal/model"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	results := []model.Result{{
		InputURL:   "example.com",
		URL:        "https://example.com",
		FinalURL:   "https://www.example.com/",
		StatusCode: 200,
		Title:      "Example",
		Technologies: []model.Technology{
			{Name: "Nginx", Version: "1.25"},
		},
		DurationMS: 12,
		ScannedAt:  time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
	}}

	tests := []struct {
		format   string
		contains string
	}{
		{format: "jsonl", contains: `"input_url":"example.com"`},
		{format: "json", contains: `"technologies": [`},
		{format: "csv", contains: "input_url,url,final_url"},
		{format: "table", contains: "STATUS"},
		{format: "markdown", contains: "| Status | URL |"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.format, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			if err := Write(&output, test.format, results); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
			if !strings.Contains(output.String(), test.contains) {
				t.Fatalf("Write() output %q does not contain %q", output.String(), test.contains)
			}
			if test.format == "jsonl" {
				var decoded model.Result
				if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
					t.Fatalf("JSONL is invalid: %v", err)
				}
			}
		})
	}
}

func TestWriteAliasesAndInvalidFormat(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"ndjson", "md"} {
		var output bytes.Buffer
		if err := Write(&output, format, []model.Result{}); err != nil {
			t.Fatalf("Write(%q) error = %v", format, err)
		}
	}
	if err := Write(&bytes.Buffer{}, "xml", nil); err == nil {
		t.Fatal("Write(xml) error = nil")
	}
}
