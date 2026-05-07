package api

import "testing"

func TestParseNextLink(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "empty header",
			header: "",
			want:   "",
		},
		{
			name: "next and last",
			header: `<https://x/issues?page=2>; rel="next", ` +
				`<https://x/issues?page=5>; rel="last"`,
			want: "https://x/issues?page=2",
		},
		{
			name: "prev and first only — no next means we're done",
			header: `<https://x/issues?page=4>; rel="prev", ` +
				`<https://x/issues?page=1>; rel="first"`,
			want: "",
		},
		{
			name: "first prev next last all present",
			header: `<https://x/?page=1>; rel="first", ` +
				`<https://x/?page=2>; rel="prev", ` +
				`<https://x/?page=4>; rel="next", ` +
				`<https://x/?page=10>; rel="last"`,
			want: "https://x/?page=4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseNextLink(tt.header); got != tt.want {
				t.Errorf("parseNextLink(%q) = %q; want %q", tt.header, got, tt.want)
			}
		})
	}
}
