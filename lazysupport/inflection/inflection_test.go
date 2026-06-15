package inflection

import "testing"

func TestPluralize(t *testing.T) {
	tests := []struct {
		singular string
		want     string
	}{
		{singular: "post", want: "posts"},
		{singular: "category", want: "categories"},
		{singular: "news", want: "news"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			if got := Pluralize(tt.singular); got != tt.want {
				t.Fatalf("Pluralize(%q) = %q, want %q", tt.singular, got, tt.want)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		plural string
		want   string
	}{
		{plural: "posts", want: "post"},
		{plural: "categories", want: "category"},
		{plural: "news", want: "new"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			if got := Singularize(tt.plural); got != tt.want {
				t.Fatalf("Singularize(%q) = %q, want %q", tt.plural, got, tt.want)
			}
		})
	}
}
