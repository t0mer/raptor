// Package netguard restricts outbound HTTP destinations to mitigate SSRF. It is
// shared by every feature that makes server-side requests on behalf of a token
// owner: Custom Actions (http_request, script), request replay, and schedule
// monitoring. The deny-list is enforced against the *resolved* IP at dial time
// (defeating DNS-rebind and IP-encoding bypasses) and across redirects.
package netguard

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// Guard holds the allow/deny policy. When allow is non-empty it is a strict
// allow-list of host suffixes; deny suffixes/CIDRs always block. By default,
// requests to internal targets (loopback, link-local incl. cloud metadata,
// private and CGNAT ranges) are blocked unless allowInternal is set.
type Guard struct {
	allow         []string
	deny          []string
	allowInternal bool
}

// New builds a Guard.
func New(allow, deny []string, allowInternal bool) *Guard {
	return &Guard{allow: normalize(allow), deny: normalize(deny), allowInternal: allowInternal}
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

// Client builds an HTTP client that enforces the guard: the policy is checked
// against the resolved IP at dial time and every redirect target is re-validated.
func (g *Guard) Client(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				host = address
			}
			return g.checkIP(net.ParseIP(host))
		},
	}
	transport := &http.Transport{
		// No environment proxy: a proxy would make the dial-time IP check
		// validate the proxy rather than the real target.
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return g.Check(req.URL.String())
		},
	}
}

// checkIP blocks a connection whose resolved IP matches a deny entry or, unless
// allowInternal is set, targets an internal range.
func (g *Guard) checkIP(ip net.IP) error {
	if ip == nil {
		return nil
	}
	for _, d := range g.deny {
		if ipMatchesPattern(ip, d) {
			return fmt.Errorf("resolved IP %s is denied", ip)
		}
	}
	if !g.allowInternal && isInternalIP(ip) {
		return fmt.Errorf("resolved IP %s is internal (blocked; set --action-allow-internal to permit)", ip)
	}
	return nil
}

// CheckIP exposes the resolved-IP policy check (for callers that resolve first).
func (g *Guard) CheckIP(ip net.IP) error { return g.checkIP(ip) }

// cgnat is the RFC 6598 shared address space (100.64.0.0/10), used by some cloud
// metadata endpoints (e.g. Alibaba 100.100.100.200) and carrier-grade NAT.
var _, cgnat, _ = net.ParseCIDR("100.64.0.0/10")

// isInternalIP reports whether an IP is loopback, link-local (incl. the cloud
// metadata 169.254.169.254), private (RFC1918 / ULA), CGNAT, unspecified or
// multicast.
func isInternalIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() ||
		cgnat.Contains(ip)
}

func ipMatchesPattern(ip net.IP, pattern string) bool {
	if _, cidr, err := net.ParseCIDR(pattern); err == nil {
		return cidr.Contains(ip)
	}
	if pip := net.ParseIP(pattern); pip != nil {
		return pip.Equal(ip)
	}
	return false
}

// Check validates a target URL (scheme + host policy) before a request is made.
func (g *Guard) Check(rawURL string) error {
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
