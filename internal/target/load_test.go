package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		explicit   []string
		positional []string
		input      string
		want       []string
	}{
		{
			name:       "combines and deduplicates",
			explicit:   []string{"https://one.test", "https://same.test"},
			positional: []string{"https://three.test", "https://same.test"},
			input:      "# ignored\n\nhttps://two.test\nhttps://same.test\n",
			want:       []string{"https://one.test", "https://same.test", "https://two.test", "https://three.test"},
		},
		{
			name:  "trims whitespace and bom",
			input: "\ufeffexample.com  \n  second.test\n",
			want:  []string{"example.com", "second.test"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := Load(test.explicit, test.positional, "-", strings.NewReader(test.input))
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if strings.Join(got, "|") != strings.Join(test.want, "|") {
				t.Fatalf("Load() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "targets.txt")
	if err := os.WriteFile(path, []byte("one.test\ntwo.test\n"), 0o600); err != nil {
		t.Fatalf("write target list: %v", err)
	}
	got, err := Load(nil, nil, path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if strings.Join(got, ",") != "one.test,two.test" {
		t.Fatalf("Load() = %q", got)
	}

	if _, err := Load(nil, nil, path+".missing", strings.NewReader("")); err == nil {
		t.Fatal("Load(missing) error = nil")
	}
}
