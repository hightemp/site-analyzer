package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"site-analyzer/internal/model"
)

func TestParseGroupMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    GroupMode
		wantErr bool
	}{
		{name: "none", value: "", want: GroupNone},
		{name: "cms", value: "cms", want: GroupCMS},
		{name: "normalized", value: " CMS ", want: GroupCMS},
		{name: "unsupported", value: "category", wantErr: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseGroupMode(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseGroupMode() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("ParseGroupMode() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGroupByCMS(t *testing.T) {
	t.Parallel()

	groups := GroupByCMS(cmsTestResults())
	if len(groups) != 3 {
		t.Fatalf("GroupByCMS() returned %d groups, want 3: %+v", len(groups), groups)
	}

	wantNames := []string{"Joomla", "WordPress", "Unknown"}
	gotNames := make([]string, 0, len(groups))
	for _, group := range groups {
		gotNames = append(gotNames, group.CMS)
		if group.SiteCount != len(group.Sites) {
			t.Errorf("group %q site_count = %d, sites = %d", group.CMS, group.SiteCount, len(group.Sites))
		}
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("group names = %q, want %q", gotNames, wantNames)
	}

	wantSites := map[string][]string{
		"Joomla":    {"two.test"},
		"WordPress": {"one.test", "two.test", "five.test"},
		"Unknown":   {"three.test", "four.test"},
	}
	for _, group := range groups {
		gotSites := make([]string, 0, len(group.Sites))
		for _, site := range group.Sites {
			gotSites = append(gotSites, site.InputURL)
		}
		if !reflect.DeepEqual(gotSites, wantSites[group.CMS]) {
			t.Errorf("group %q sites = %q, want %q", group.CMS, gotSites, wantSites[group.CMS])
		}
	}

	if groups := GroupByCMS(nil); groups == nil || len(groups) != 0 {
		t.Fatalf("GroupByCMS(nil) = %#v, want non-nil empty slice", groups)
	}
}

func TestWriteCMSFormats(t *testing.T) {
	t.Parallel()

	results := cmsTestResults()
	tests := []struct {
		format   string
		contains string
	}{
		{format: "jsonl", contains: `"cms":"WordPress"`},
		{format: "json", contains: `"site_count": 3`},
		{format: "csv", contains: "cms,site_count,input_url"},
		{format: "table", contains: "CMS"},
		{format: "markdown", contains: "| CMS | Count |"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.format, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			if err := Write(&output, test.format, results, Options{GroupBy: GroupCMS}); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
			for _, required := range []string{test.contains, "WordPress", "one.test", "Unknown"} {
				if !strings.Contains(output.String(), required) {
					t.Fatalf("Write() output %q does not contain %q", output.String(), required)
				}
			}
		})
	}
}

func TestWriteCMSJSONAndCSVShape(t *testing.T) {
	t.Parallel()

	results := cmsTestResults()
	var jsonOutput bytes.Buffer
	if err := Write(&jsonOutput, "json", results, Options{GroupBy: GroupCMS}); err != nil {
		t.Fatalf("Write(json) error = %v", err)
	}
	var groups []model.CMSGroup
	if err := json.Unmarshal(jsonOutput.Bytes(), &groups); err != nil {
		t.Fatalf("decode grouped JSON: %v", err)
	}
	if len(groups) != 3 || groups[1].CMS != "WordPress" || groups[1].Sites[0].Technologies[0].Version != "6.5" {
		t.Fatalf("grouped JSON did not preserve full results: %+v", groups)
	}

	var jsonlOutput bytes.Buffer
	if err := Write(&jsonlOutput, "jsonl", results, Options{GroupBy: GroupCMS}); err != nil {
		t.Fatalf("Write(jsonl) error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(jsonlOutput.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("grouped JSONL lines = %d, want 3", len(lines))
	}
	for index, line := range lines {
		var group model.CMSGroup
		if err := json.Unmarshal([]byte(line), &group); err != nil {
			t.Fatalf("decode grouped JSONL line %d: %v", index, err)
		}
		if group.CMS == "" || group.SiteCount != len(group.Sites) {
			t.Fatalf("invalid grouped JSONL line %d: %+v", index, group)
		}
	}

	var csvOutput bytes.Buffer
	if err := Write(&csvOutput, "csv", results, Options{GroupBy: GroupCMS}); err != nil {
		t.Fatalf("Write(csv) error = %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(csvOutput.String())).ReadAll()
	if err != nil {
		t.Fatalf("decode grouped CSV: %v", err)
	}
	if len(records) != 7 {
		t.Fatalf("grouped CSV rows = %d, want 7", len(records))
	}
	if !reflect.DeepEqual(records[0][:3], []string{"cms", "site_count", "input_url"}) {
		t.Fatalf("grouped CSV header = %q", records[0])
	}
}

func TestWriteUngroupedShape(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	result := model.Result{InputURL: "example.test", Technologies: []model.Technology{}}
	if err := Write(&output, "jsonl", []model.Result{result}, Options{}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if strings.Contains(output.String(), `"cms"`) || !strings.HasPrefix(output.String(), `{"input_url":"example.test"`) {
		t.Fatalf("ungrouped output shape changed: %s", output.String())
	}
}

func cmsTestResults() []model.Result {
	return []model.Result{
		{
			InputURL: "one.test",
			Technologies: []model.Technology{
				{Name: "WordPress", Version: "6.5", Categories: []string{"CMS"}},
				{Name: "WordPress", Version: "6.4", Categories: []string{" cms "}},
				{Name: "Nginx", Categories: []string{"Web servers"}},
			},
		},
		{
			InputURL: "two.test",
			Technologies: []model.Technology{
				{Name: "Joomla", Version: "5", Categories: []string{"CMS"}},
				{Name: "WordPress", Version: "6.3", Categories: []string{"CMS"}},
			},
		},
		{InputURL: "three.test", Technologies: []model.Technology{{Name: "Go", Categories: []string{"Programming languages"}}}},
		{InputURL: "four.test", Technologies: []model.Technology{}, Error: "request failed"},
		{InputURL: "five.test", Technologies: []model.Technology{{Name: "wordpress", Version: "6.2", Categories: []string{"CMS"}}}},
	}
}
