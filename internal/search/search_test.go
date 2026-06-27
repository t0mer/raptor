package search

import (
	"strings"
	"testing"
	"time"
)

var anchor = time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)

func TestEmpty(t *testing.T) {
	if !Parse("", anchor).Empty() {
		t.Error("empty query should produce empty filter")
	}
	if !Parse("   ", anchor).Empty() {
		t.Error("whitespace query should produce empty filter")
	}
}

func TestFreeText(t *testing.T) {
	f := Parse("hello", anchor)
	if f.SQL != "content LIKE ?" {
		t.Errorf("SQL = %q", f.SQL)
	}
	if len(f.Args) != 1 || f.Args[0] != "%hello%" {
		t.Errorf("Args = %v", f.Args)
	}
}

func TestFieldFilters(t *testing.T) {
	cases := []struct {
		q       string
		wantSQL string
		wantArg any
	}{
		{"method:post", "method = ?", "POST"},
		{"type:Web", "type = ?", "web"},
		{"ip:1.2.3.4", "ip = ?", "1.2.3.4"},
		{"host:example", "hostname LIKE ?", "%example%"},
		{"url:/cb", "url LIKE ?", "%/cb%"},
		{"query:token", "query LIKE ?", "%token%"},
		{"content:foo", "content LIKE ?", "%foo%"},
	}
	for _, c := range cases {
		f := Parse(c.q, anchor)
		if f.SQL != c.wantSQL {
			t.Errorf("%q: SQL = %q, want %q", c.q, f.SQL, c.wantSQL)
		}
		if len(f.Args) != 1 || f.Args[0] != c.wantArg {
			t.Errorf("%q: Args = %v, want [%v]", c.q, f.Args, c.wantArg)
		}
	}
}

func TestMultipleTermsAnded(t *testing.T) {
	f := Parse("method:POST content:charge", anchor)
	if f.SQL != "method = ? AND content LIKE ?" {
		t.Errorf("SQL = %q", f.SQL)
	}
	if len(f.Args) != 2 {
		t.Fatalf("Args = %v", f.Args)
	}
}

func TestQuotedValue(t *testing.T) {
	f := Parse(`content:"hello world"`, anchor)
	if len(f.Args) != 1 || f.Args[0] != "%hello world%" {
		t.Errorf("Args = %v", f.Args)
	}
}

func TestHeadersKeyValue(t *testing.T) {
	f := Parse("headers.x-github-event:push", anchor)
	if f.SQL != "(headers LIKE ? AND headers LIKE ?)" {
		t.Errorf("SQL = %q", f.SQL)
	}
	if f.Args[0] != "%x-github-event%" || f.Args[1] != "%push%" {
		t.Errorf("Args = %v", f.Args)
	}
}

func TestExists(t *testing.T) {
	f := Parse("_exists_:custom_action_errors", anchor)
	if !strings.Contains(f.SQL, "custom_action_errors != '{}'") {
		t.Errorf("SQL = %q", f.SQL)
	}
	if len(f.Args) != 0 {
		t.Errorf("Args = %v, want none", f.Args)
	}
	// Unknown field is ignored.
	if !Parse("_exists_:secret_column", anchor).Empty() {
		t.Error("unknown _exists_ field should be ignored")
	}
}

func TestCreatedAtRange(t *testing.T) {
	f := Parse("created_at:[* TO now-14d]", anchor)
	if f.SQL != "(created_at <= ?)" {
		t.Errorf("SQL = %q", f.SQL)
	}
	want := anchor.Add(-14 * 24 * time.Hour).Format(time.RFC3339Nano)
	if len(f.Args) != 1 || f.Args[0] != want {
		t.Errorf("Args = %v, want [%v]", f.Args, want)
	}
}

func TestCreatedAtBothBounds(t *testing.T) {
	f := Parse("created_at:[2026-06-01 TO 2026-06-20]", anchor)
	if f.SQL != "(created_at >= ? AND created_at <= ?)" {
		t.Errorf("SQL = %q", f.SQL)
	}
	if len(f.Args) != 2 {
		t.Fatalf("Args = %v", f.Args)
	}
	if !strings.HasPrefix(f.Args[0].(string), "2026-06-01") {
		t.Errorf("lower bound = %v", f.Args[0])
	}
}

func TestMalformedRangeDoesNotPanic(t *testing.T) {
	// Malformed bounds must be ignored, never panic.
	for _, q := range []string{
		"created_at:[now- TO *]",
		"created_at:[now TO now+]",
		"created_at:[bogus TO bogus]",
		"created_at:[]",
		"created_at:",
	} {
		_ = Parse(q, anchor) // must not panic
	}
}

func TestNoSQLInjectionInValues(t *testing.T) {
	// A malicious value must be bound as an argument, never inlined. Quote it so
	// it stays a single token despite the spaces.
	f := Parse(`content:"'; DROP TABLE requests;--"`, anchor)
	if f.SQL != "content LIKE ?" {
		t.Errorf("SQL = %q (value must be parameterised)", f.SQL)
	}
	if len(f.Args) != 1 || !strings.Contains(f.Args[0].(string), "DROP TABLE") {
		t.Errorf("value not preserved as a single arg: %v", f.Args)
	}
}
