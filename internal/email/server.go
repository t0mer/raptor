// Package email implements an inbound SMTP server that captures messages sent to
// {token}@{email-domain} addresses, parses their MIME content and attachments,
// runs DKIM/SPF/DMARC checks, and records them against the matching token.
package email

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/emersion/go-smtp"

	"github.com/t0mer/raptor/internal/capture"
)

// DefaultMaxMessageBytes caps an inbound message size.
const DefaultMaxMessageBytes int64 = 25 << 20 // 25 MiB

// Server is the inbound SMTP capture server.
type Server struct {
	smtp     *smtp.Server
	capturer *capture.Capturer
	domain   string
	checker  Checker
	logger   *slog.Logger
}

// Option configures the Server.
type Option func(*Server)

// WithChecker overrides the message-authentication checker (tests use a no-op).
func WithChecker(c Checker) Option {
	return func(s *Server) {
		if c != nil {
			s.checker = c
		}
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		if l != nil {
			s.logger = l
		}
	}
}

// New builds an inbound SMTP server. domain is the email suffix (e.g.
// "emailhook.site") used to extract the token from the recipient address.
func New(capturer *capture.Capturer, domain string, opts ...Option) *Server {
	s := &Server{
		capturer: capturer,
		domain:   domain,
		checker:  realChecks,
		logger:   slog.Default(),
	}
	for _, o := range opts {
		o(s)
	}

	be := smtp.BackendFunc(func(c *smtp.Conn) (smtp.Session, error) {
		return s.newSession(c), nil
	})
	srv := smtp.NewServer(be)
	srv.Domain = domain
	srv.MaxMessageBytes = DefaultMaxMessageBytes
	srv.MaxRecipients = 100
	srv.ReadTimeout = 60 * time.Second
	srv.WriteTimeout = 60 * time.Second
	srv.AllowInsecureAuth = true // capture server: no auth, plaintext inbound
	s.smtp = srv
	return s
}

// ListenAndServe binds the given address and serves until Shutdown/Close.
func (s *Server) ListenAndServe(addr string) error {
	s.smtp.Addr = addr
	s.logger.Info("smtp capture listening", "addr", addr, "domain", s.domain)
	return s.smtp.ListenAndServe()
}

// Serve serves on an existing listener (used by tests).
func (s *Server) Serve(l net.Listener) error { return s.smtp.Serve(l) }

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.smtp.Shutdown(ctx)
}
