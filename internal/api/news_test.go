package api

import (
	"testing"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello world</p>", "Hello world\n"},
		{"<div>One</div><div>Two</div>", "One\nTwo\n"},
		{"<a href=\"#\">Link</a>", "Link"},
		{"Line<br>Break", "Line\nBreak"},
		{"<ul><li>Item 1</li><li>Item 2</li></ul>", "Item 1\nItem 2\n"},
		{"No tags", "No tags"},
		{"", ""},
		{"Invalid <b>HTML", "Invalid HTML"},
	}

	for _, tt := range tests {
		if got := stripHTML(tt.input); got != tt.want {
			t.Errorf("stripHTML(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello   world  ", "Hello   world"},
		{"Line 1\n\n\nLine 2", "Line 1\n\nLine 2"},
		{"  Trim spaces  ", "Trim spaces"},
		{"\n\nLeading and trailing\n\n", "Leading and trailing"},
		{"Multiple\n\n\n\nBlank\n\nLines", "Multiple\n\nBlank\n\nLines"},
	}

	for _, tt := range tests {
		if got := normalizeText(tt.input); got != tt.want {
			t.Errorf("normalizeText(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}
