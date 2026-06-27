// Package dns implements an inbound DNS server that captures queries sent to
// {token}.{dns-domain} (and any subdomain thereof), records them against the
// matching token, and returns a minimal response.
package dns

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/miekg/dns"

	"github.com/t0mer/raptor/internal/capture"
	"github.com/t0mer/raptor/internal/models"
)

// Server is the inbound DNS capture server (UDP + TCP).
type Server struct {
	capturer *capture.Capturer
	domain   string // e.g. "dnshook.site"
	answerA  net.IP // A-record answer returned for captured queries
	logger   *slog.Logger

	udp *dns.Server
	tcp *dns.Server
}

// Option configures the Server.
type Option func(*Server)

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		if l != nil {
			s.logger = l
		}
	}
}

// WithAnswerA sets the IPv4 address returned for captured A queries.
func WithAnswerA(ip net.IP) Option {
	return func(s *Server) { s.answerA = ip }
}

// New builds a DNS capture server. domain is the suffix (e.g. "dnshook.site")
// under which the label adjacent to the suffix identifies the token.
func New(capturer *capture.Capturer, domain string, opts ...Option) *Server {
	s := &Server{
		capturer: capturer,
		domain:   strings.ToLower(strings.TrimSuffix(domain, ".")),
		answerA:  net.IPv4(127, 0, 0, 1),
		logger:   slog.Default(),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ListenAndServe starts UDP and TCP listeners on addr.
func (s *Server) ListenAndServe(addr string) error {
	h := dns.HandlerFunc(s.handle)
	s.udp = &dns.Server{Addr: addr, Net: "udp", Handler: h}
	s.tcp = &dns.Server{Addr: addr, Net: "tcp", Handler: h}
	s.logger.Info("dns capture listening", "addr", addr, "domain", s.domain)

	errc := make(chan error, 2)
	go func() { errc <- s.udp.ListenAndServe() }()
	go func() { errc <- s.tcp.ListenAndServe() }()
	return <-errc
}

// ServePacket serves DNS on an existing UDP connection (used by tests).
func (s *Server) ServePacket(conn net.PacketConn) error {
	s.udp = &dns.Server{PacketConn: conn, Handler: dns.HandlerFunc(s.handle)}
	return s.udp.ActivateAndServe()
}

// Shutdown stops both listeners.
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	if s.udp != nil {
		if e := s.udp.ShutdownContext(ctx); e != nil {
			err = e
		}
	}
	if s.tcp != nil {
		if e := s.tcp.ShutdownContext(ctx); e != nil {
			err = e
		}
	}
	return err
}

func (s *Server) handle(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	if len(r.Question) > 0 {
		q := r.Question[0]
		if tok := s.resolveToken(q.Name); tok != nil {
			s.record(tok, q, w.RemoteAddr())
			s.answer(m, q)
		} else {
			m.Rcode = dns.RcodeNameError // NXDOMAIN for unknown tokens
		}
	}
	_ = w.WriteMsg(m)
}

// resolveToken extracts the token label (the one adjacent to the configured
// domain suffix) from a query name and resolves it.
func (s *Server) resolveToken(qname string) *models.Token {
	name := strings.ToLower(strings.TrimSuffix(qname, "."))
	suffix := "." + s.domain
	if name == s.domain || !strings.HasSuffix(name, suffix) {
		return nil
	}
	prefix := strings.TrimSuffix(name, suffix)
	label := prefix
	if i := strings.LastIndex(prefix, "."); i >= 0 {
		label = prefix[i+1:]
	}
	if label == "" {
		return nil
	}
	tok, err := s.capturer.Resolve(context.Background(), label)
	if err != nil {
		return nil
	}
	return tok
}

func (s *Server) record(tok *models.Token, q dns.Question, remote net.Addr) {
	ip := ""
	if host, _, err := net.SplitHostPort(remote.String()); err == nil {
		ip = host
	}
	name := strings.TrimSuffix(q.Name, ".")
	now := time.Now().UTC()
	req := &models.Request{
		UUID:     uuid.NewString(),
		TokenID:  tok.UUID,
		Type:     models.RequestTypeDNS,
		Method:   dns.TypeToString[q.Qtype],
		IP:       ip,
		Hostname: name,
		Content:  name,
		Query: map[string][]string{
			"name":  {name},
			"type":  {dns.TypeToString[q.Qtype]},
			"class": {dns.ClassToString[q.Qclass]},
		},
		Sorting:   now.UnixMilli(),
		CreatedAt: now,
	}
	if err := s.capturer.Record(context.Background(), tok, req); err != nil {
		s.logger.Warn("record dns failed", "token", tok.UUID, "error", err)
	}
}

// answer adds a minimal answer: an A record for A queries, otherwise NOERROR
// with no answer section.
func (s *Server) answer(m *dns.Msg, q dns.Question) {
	if q.Qtype == dns.TypeA && s.answerA != nil {
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   s.answerA,
		}
		m.Answer = append(m.Answer, rr)
	}
}
