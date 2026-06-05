package components

import (
	"testing"

	"github.com/nqui/vault-tui/internal/vault"
)

func TestSearchFilter(t *testing.T) {
	entries := []vault.PathEntry{
		{Name: "APP1/", IsDir: true},
		{Name: "APP2/", IsDir: true},
		{Name: "APP3/", IsDir: true},
		{Name: "shared", IsDir: false},
	}

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty matches all", "", []string{"APP1/", "APP2/", "APP3/", "shared"}},
		{"prefix app lowercase", "app", []string{"APP1/", "APP2/", "APP3/"}},
		{"prefix exact one", "APP1", []string{"APP1/"}},
		{"case insensitive", "ApP2", []string{"APP2/"}},
		{"leaf prefix", "sh", []string{"shared"}},
		{"no match", "zzz", nil},
		{"non-prefix substring no match", "1", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSearch()
			m.SetResults(entries)
			m.input.SetValue(tt.query)
			m.applyFilter()

			got := make([]string, 0, len(m.filtered))
			for _, e := range m.filtered {
				got = append(got, e.Name)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("query %q: got %v, want %v", tt.query, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("query %q: got %v, want %v", tt.query, got, tt.want)
				}
			}
		})
	}
}
