package dns

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/miekg/dns"

	"github.com/t0mer/raptor/internal/capture"
	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/store"
)

func TestResolveToken(t *testing.T) {
	c, st := setup(t)
	tok := &models.Token{UUID: uuid.NewString(), Alias: "myhook", Premium: true}
	if err := st.CreateToken(context.Background(), tok); err != nil {
		t.Fatal(err)
	}
	s := New(c, "dnshook.site")

	cases := map[string]bool{
		tok.UUID + ".dnshook.site.":              true,
		"foo.bar." + tok.UUID + ".dnshook.site.": true,
		"myhook.dnshook.site.":                   true, // alias
		"unknown.dnshook.site.":                  false,
		"dnshook.site.":                          false,
		tok.UUID + ".other.site.":                false,
	}
	for name, want := range cases {
		got := s.resolveToken(name) != nil
		if got != want {
			t.Errorf("resolveToken(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestInboundDNSCaptured(t *testing.T) {
	c, st := setup(t)
	tok := &models.Token{UUID: uuid.NewString(), Premium: true}
	if err := st.CreateToken(context.Background(), tok); err != nil {
		t.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	s := New(c, "dnshook.site")
	go s.ServePacket(conn) //nolint:errcheck
	defer s.Shutdown(context.Background())

	addr := conn.LocalAddr().String()
	m := new(dns.Msg)
	qname := "probe." + tok.UUID + ".dnshook.site."
	m.SetQuestion(qname, dns.TypeA)

	resp, err := dns.Exchange(m, addr)
	if err != nil {
		t.Fatalf("dns exchange: %v", err)
	}
	if len(resp.Answer) == 0 {
		t.Error("expected an A answer for a known token")
	}

	var reqs []*models.Request
	for i := 0; i < 50; i++ {
		reqs, _ = st.ListRequests(context.Background(), tok.UUID, 10, 0)
		if len(reqs) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 captured DNS query, got %d", len(reqs))
	}
	got := reqs[0]
	if got.Type != models.RequestTypeDNS || got.Method != "A" {
		t.Errorf("captured dns mismatch: type=%q method=%q", got.Type, got.Method)
	}
	if got.Content != "probe."+tok.UUID+".dnshook.site" {
		t.Errorf("content = %q", got.Content)
	}
}

func TestUnknownTokenNXDOMAIN(t *testing.T) {
	c, _ := setup(t)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	s := New(c, "dnshook.site")
	go s.ServePacket(conn) //nolint:errcheck
	defer s.Shutdown(context.Background())

	m := new(dns.Msg)
	m.SetQuestion("nope.dnshook.site.", dns.TypeA)
	resp, err := dns.Exchange(m, conn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Rcode != dns.RcodeNameError {
		t.Errorf("Rcode = %d, want NXDOMAIN", resp.Rcode)
	}
}

func setup(t *testing.T) (*capture.Capturer, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return capture.New(st, "http://x"), st
}
