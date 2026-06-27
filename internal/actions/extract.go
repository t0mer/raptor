package actions

import (
	"context"
	"fmt"
	"regexp"

	"github.com/tidwall/gjson"

	"github.com/t0mer/raptor/internal/models"
)

func init() {
	register("extract_json", newExtractJSON)
	register("extract_regex", newExtractRegex)
}

// extract_json — read a value from a JSON source via a gjson path and store it
// in a variable. The source defaults to the request body.
type extractJSON struct {
	source   string
	path     string
	variable string
}

func newExtractJSON(a *models.Action) (runner, error) {
	src := strParam(a, "source")
	if src == "" {
		src = "$request.content$"
	}
	path := strParam(a, "path")
	if path == "" {
		return nil, fmt.Errorf("extract_json: path is required")
	}
	return &extractJSON{source: src, path: path, variable: strParam(a, "variable")}, nil
}

func (e *extractJSON) run(_ context.Context, ec *ExecContext) error {
	src := ec.Interp(e.source)
	res := gjson.Get(src, e.path)
	if !res.Exists() {
		ec.Logf("extract_json: path %q not found", e.path)
		return nil
	}
	if e.variable != "" {
		ec.SetVar(e.variable, res.String())
	}
	ec.Logf("extract_json: %s = %s", e.variable, res.String())
	return nil
}

// extract_regex — capture a regex group from a source into a variable.
type extractRegex struct {
	source   string
	re       *regexp.Regexp
	group    int
	variable string
}

func newExtractRegex(a *models.Action) (runner, error) {
	src := strParam(a, "source")
	if src == "" {
		src = "$request.content$"
	}
	pattern := strParam(a, "pattern")
	if pattern == "" {
		return nil, fmt.Errorf("extract_regex: pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("extract_regex: invalid pattern: %w", err)
	}
	return &extractRegex{source: src, re: re, group: intParam(a, "group", 1), variable: strParam(a, "variable")}, nil
}

func (e *extractRegex) run(_ context.Context, ec *ExecContext) error {
	src := ec.Interp(e.source)
	m := e.re.FindStringSubmatch(src)
	if m == nil || e.group >= len(m) {
		ec.Logf("extract_regex: no match")
		return nil
	}
	val := m[e.group]
	if e.variable != "" {
		ec.SetVar(e.variable, val)
	}
	ec.Logf("extract_regex: %s = %s", e.variable, val)
	return nil
}
