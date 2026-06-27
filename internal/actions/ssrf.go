package actions

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ssrfGuard restricts which hosts outbound actions (http_request, script) may
// reach. When allow is non-empty it is a strict allow-list of host suffixes;
// deny host suffixes always block. With both empty, all hosts are permitted
// (self-hosters opt in via --action-allow / --action-deny).
type ssrfGuard struct {
	allow []string
	deny  []string
}

func newSSRFGuard(allow, deny []string) *ssrfGuard {
	return &ssrfGuard{allow: normalize(allow), deny: normalize(deny)}
}

func normalize(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// check validates a target URL against the guard, returning an error if blocked.
func (g *ssrfGuard) check(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme %q not allowed", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("missing host")
	}
	for _, d := range g.deny {
		if hostMatches(host, d) {
			return fmt.Errorf("host %q is denied", host)
		}
	}
	if len(g.allow) > 0 {
		for _, a := range g.allow {
			if hostMatches(host, a) {
				return nil
			}
		}
		return fmt.Errorf("host %q is not in the allow-list", host)
	}
	return nil
}

// hostMatches reports whether host equals or is a subdomain of pattern, or (when
// pattern is an IP/CIDR) falls within it.
func hostMatches(host, pattern string) bool {
	if host == pattern || strings.HasSuffix(host, "."+pattern) {
		return true
	}
	if _, cidr, err := net.ParseCIDR(pattern); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return cidr.Contains(ip)
		}
	}
	return false
}
