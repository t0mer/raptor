package email

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
)

// recipient pairs a captured RCPT TO address with its resolved token.
type recipient struct {
	address string
	token   *models.Token
}

// session implements smtp.Session for one inbound connection.
type session struct {
	srv      *Server
	remoteIP net.IP
	helo     string
	mailFrom string
	rcpts    []recipient
}

func (s *Server) newSession(c *smtp.Conn) *session {
	var ip net.IP
	if c.Conn() != nil {
		if host, _, err := net.SplitHostPort(c.Conn().RemoteAddr().String()); err == nil {
			ip = net.ParseIP(host)
		}
	}
	return &session{srv: s, remoteIP: ip, helo: c.Hostname()}
}

func (s *session) Reset() {
	s.mailFrom = ""
	s.rcpts = nil
}

func (s *session) Logout() error { return nil }

func (s *session) Mail(from string, _ *smtp.MailOptions) error {
	s.mailFrom = from
	return nil
}

// Rcpt resolves the recipient to a token. Addresses outside the configured email
// domain, or with no matching token, are rejected so the server is not an open
// sink.
func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	local, domain := splitAddress(to)
	if domain != "" && !strings.EqualFold(domain, s.srv.domain) {
		return &smtp.SMTPError{Code: 550, Message: "relay not permitted"}
	}
	// Strip subaddressing (token+tag@...).
	if i := strings.IndexByte(local, '+'); i >= 0 {
		local = local[:i]
	}
	tok, err := s.srv.capturer.Resolve(context.Background(), local)
	if err != nil {
		return &smtp.SMTPError{Code: 550, Message: "no such mailbox"}
	}
	s.rcpts = append(s.rcpts, recipient{address: to, token: tok})
	return nil
}

// Data reads the full message, parses it once, runs auth checks once, and
// records one captured request per recipient token (with attachments).
func (s *session) Data(r io.Reader) error {
	if len(s.rcpts) == 0 {
		return &smtp.SMTPError{Code: 554, Message: "no valid recipients"}
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	parsed, err := parseMessage(raw)
	if err != nil {
		// Still capture the raw message even if MIME parsing failed.
		parsed = &parsedEmail{Headers: map[string][]string{}, Text: string(raw)}
	}

	checks := s.srv.checker(raw, s.remoteIP, s.helo, envelopeAddress(s.mailFrom))
	now := time.Now().UTC()

	for _, rc := range s.rcpts {
		req := s.buildRequest(rc, parsed, raw, checks, now)
		if err := s.srv.capturer.Record(context.Background(), rc.token, req); err != nil {
			if errors.Is(err, errSkip) {
				continue
			}
			s.srv.logger.Warn("record email failed", "token", rc.token.UUID, "error", err)
			continue
		}
		s.saveAttachments(req.UUID, parsed)
	}
	return nil
}

func (s *session) buildRequest(rc recipient, parsed *parsedEmail, raw []byte, checks map[string]any, now time.Time) *models.Request {
	sender := parsed.From
	if sender == "" {
		sender = envelopeAddress(s.mailFrom)
	}
	ip := ""
	if s.remoteIP != nil {
		ip = s.remoteIP.String()
	}
	return &models.Request{
		UUID:         uuid.NewString(),
		TokenID:      rc.token.UUID,
		Type:         models.RequestTypeEmail,
		IP:           ip,
		Hostname:     s.helo,
		Sender:       sender,
		MessageID:    parsed.MessageID,
		Destinations: rc.address,
		Subject:      parsed.Subject,
		Content:      parsed.displayContent(),
		TextContent:  parsed.Text,
		Headers:      parsed.Headers,
		Checks:       cloneChecks(checks),
		Size:         len(raw),
		Sorting:      now.UnixMilli(),
		CreatedAt:    now,
	}
}

func (s *session) saveAttachments(requestID string, parsed *parsedEmail) {
	for _, f := range parsed.Files {
		if _, err := s.srv.capturer.SaveFile(context.Background(), requestID, f.Filename, f.ContentType, f.Data); err != nil {
			s.srv.logger.Warn("save attachment failed", "filename", f.Filename, "error", err)
		}
	}
}

// errSkip is a placeholder for future "dont_save" handling.
var errSkip = errors.New("skip")

func splitAddress(addr string) (local, domain string) {
	addr = strings.Trim(addr, "<>")
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		return addr[:i], addr[i+1:]
	}
	return addr, ""
}

func envelopeAddress(s string) string { return strings.Trim(s, "<>") }

func cloneChecks(c map[string]any) map[string]any {
	out := make(map[string]any, len(c))
	for k, v := range c {
		out[k] = v
	}
	return out
}
