package actions

import (
	"regexp"
	"strings"

	"github.com/t0mer/raptor/internal/models"
)

// varRef matches $name$ references, where name may contain dotted paths
// (e.g. $request.query.id$ or $my_var$).
var varRef = regexp.MustCompile(`\$([a-zA-Z0-9_.\-]+)\$`)

func interpolate(ec *ExecContext, s string) string {
	if !strings.Contains(s, "$") {
		return s
	}
	return varRef.ReplaceAllStringFunc(s, func(m string) string {
		name := m[1 : len(m)-1]
		return resolveVar(ec, name)
	})
}

// resolveVar resolves a variable name. Names prefixed with "request." expose
// fields of the captured request; everything else reads the variables map.
func resolveVar(ec *ExecContext, name string) string {
	if v, ok := ec.Vars[name]; ok {
		return v
	}
	if rest, ok := strings.CutPrefix(name, "request."); ok && ec.Request != nil {
		return requestField(ec.Request, rest)
	}
	return ""
}

func requestField(r *models.Request, field string) string {
	switch {
	case field == "content" || field == "body":
		return r.Content
	case field == "method":
		return r.Method
	case field == "ip":
		return r.IP
	case field == "hostname":
		return r.Hostname
	case field == "subject":
		return r.Subject
	case field == "sender":
		return r.Sender
	case strings.HasPrefix(field, "query."):
		if vals, ok := r.Query[strings.TrimPrefix(field, "query.")]; ok && len(vals) > 0 {
			return vals[0]
		}
	case strings.HasPrefix(field, "header."):
		return headerValue(r.Headers, strings.TrimPrefix(field, "header."))
	}
	return ""
}

// headerValue does a case-insensitive header lookup.
func headerValue(headers map[string][]string, key string) string {
	for k, vals := range headers {
		if strings.EqualFold(k, key) && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}
