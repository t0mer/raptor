package email

import (
	"context"
	"net"
	"net/smtp"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/capture"
	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/store"
)

func TestParseMessage(t *testing.T) {
	raw := []byte("From: Alice <alice@example.com>\r\n" +
		"Subject: Hello there\r\n" +
		"Message-ID: <abc@example.com>\r\n" +
		"Content-Type: text/plain\r\n\r\n" +
		"body text\r\n")
	p, err := parseMessage(raw)
	if err != nil {
		t.Fatalf("parseMessage: %v", err)
	}
	if p.Subject != "Hello there" {
		t.Errorf("subject = %q", p.Subject)
	}
	if p.From != "alice@example.com" {
		t.Errorf("from = %q", p.From)
	}
	if p.Text == "" {
		t.Errorf("text body empty")
	}
}

func TestSplitAddress(t *testing.T) {
	l, d := splitAddress("<tok123@emailhook.site>")
	if l != "tok123" || d != "emailhook.site" {
		t.Errorf("split = %q / %q", l, d)
	}
}

func newCapturer(t *testing.T) (*capture.Capturer, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "e.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	c := capture.New(st, "http://x", capture.WithFilesDir(filepath.Join(t.TempDir(), "files")))
	return c, st
}

func TestInboundEmailCaptured(t *testing.T) {
	c, st := newCapturer(t)
	tok := &models.Token{UUID: uuid.NewString(), Premium: true}
	if err := st.CreateToken(context.Background(), tok); err != nil {
		t.Fatal(err)
	}

	noop := func(_ []byte, _ net.IP, _, _ string) map[string]any {
		return map[string]any{"dkim": "none", "spf": "none", "dmarc": "none"}
	}
	srv := New(c, "emailhook.site", WithChecker(noop))

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(l) //nolint:errcheck
	defer srv.Shutdown(context.Background())

	msg := []byte("From: sender@example.com\r\n" +
		"Subject: Invoice 42\r\n" +
		"Content-Type: text/plain\r\n\r\n" +
		"please pay\r\n")
	to := tok.UUID + "@emailhook.site"
	if err := smtp.SendMail(l.Addr().String(), nil, "sender@example.com", []string{to}, msg); err != nil {
		t.Fatalf("SendMail: %v", err)
	}

	// Give the server a moment to record.
	var reqs []*models.Request
	for i := 0; i < 50; i++ {
		reqs, _ = st.ListRequests(context.Background(), tok.UUID, 10, 0)
		if len(reqs) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 captured email, got %d", len(reqs))
	}
	got := reqs[0]
	if got.Type != models.RequestTypeEmail || got.Subject != "Invoice 42" {
		t.Errorf("captured email mismatch: %+v", got)
	}
	if got.Sender != "sender@example.com" {
		t.Errorf("sender = %q", got.Sender)
	}
	if got.Checks["dkim"] != "none" {
		t.Errorf("checks = %+v", got.Checks)
	}
}

func TestRcptRejectsWrongDomainAndUnknownToken(t *testing.T) {
	c, _ := newCapturer(t)
	srv := New(c, "emailhook.site")
	sess := &session{srv: srv}

	if err := sess.Rcpt("user@wrong.example", nil); err == nil {
		t.Error("expected rejection for wrong domain")
	}
	if err := sess.Rcpt("nonexistent@emailhook.site", nil); err == nil {
		t.Error("expected rejection for unknown token")
	}
}
