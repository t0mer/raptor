package api

import "testing"

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"plain":           "plain",
		"=cmd()":          "'=cmd()",
		"+1":              "'+1",
		"-1":              "'-1",
		"@SUM":            "'@SUM",
		"\tTAB":           "'\tTAB",
		"normal=embedded": "normal=embedded",
		"GET":             "GET",
	}
	for in, want := range cases {
		if got := csvSafe(in); got != want {
			t.Errorf("csvSafe(%q) = %q, want %q", in, got, want)
		}
	}
}
