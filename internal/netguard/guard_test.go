package netguard

import (
	"net"
	"testing"
)

func TestSSRFCheckScheme(t *testing.T) {
	g := New(nil, nil, true)
	for _, bad := range []string{"file:///etc/passwd", "gopher://x/", "ftp://h/"} {
		if err := g.Check(bad); err == nil {
			t.Errorf("scheme should be blocked: %s", bad)
		}
	}
	if err := g.Check("https://example.com/x"); err != nil {
		t.Errorf("https should be allowed: %v", err)
	}
}

func TestSSRFInternalRangesBlocked(t *testing.T) {
	g := New(nil, nil, false)
	internal := []string{
		"127.0.0.1",       // loopback
		"169.254.169.254", // link-local / metadata
		"10.1.2.3",        // private
		"192.168.0.1",     // private
		"172.16.0.1",      // private
		"100.100.100.200", // CGNAT (Alibaba metadata)
		"0.0.0.0",         // unspecified
		"::1",             // IPv6 loopback
	}
	for _, s := range internal {
		if err := g.CheckIP(net.ParseIP(s)); err == nil {
			t.Errorf("internal IP %s should be blocked by default", s)
		}
	}
	// Public IP allowed.
	if err := g.CheckIP(net.ParseIP("8.8.8.8")); err != nil {
		t.Errorf("public IP should be allowed: %v", err)
	}
}

func TestSSRFAllowInternal(t *testing.T) {
	g := New(nil, nil, true)
	if err := g.CheckIP(net.ParseIP("127.0.0.1")); err != nil {
		t.Errorf("allowInternal should permit loopback: %v", err)
	}
}

func TestSSRFDenyByResolvedIP(t *testing.T) {
	g := New(nil, []string{"203.0.113.0/24"}, true)
	if err := g.CheckIP(net.ParseIP("203.0.113.5")); err == nil {
		t.Error("deny CIDR should block matching IP")
	}
	if err := g.CheckIP(net.ParseIP("8.8.8.8")); err != nil {
		t.Errorf("non-denied IP should pass: %v", err)
	}
}
