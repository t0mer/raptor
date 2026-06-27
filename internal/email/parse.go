package email

import (
	"bytes"
	"io"
	"strings"

	"github.com/emersion/go-message/mail"
)

// attachment is a parsed email attachment.
type attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// parsedEmail holds the fields extracted from a raw MIME message.
type parsedEmail struct {
	From      string
	Subject   string
	MessageID string
	HTML      string
	Text      string
	Headers   map[string][]string
	Files     []attachment
}

// parseMessage extracts headers, bodies and attachments from a raw MIME message.
// It is tolerant: a malformed message still yields whatever could be read.
func parseMessage(raw []byte) (*parsedEmail, error) {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	out := &parsedEmail{Headers: map[string][]string{}}

	if subject, err := mr.Header.Subject(); err == nil {
		out.Subject = subject
	}
	if id, err := mr.Header.MessageID(); err == nil {
		out.MessageID = id
	}
	if addrs, err := mr.Header.AddressList("From"); err == nil && len(addrs) > 0 {
		out.From = addrs[0].Address
	}

	// Copy all header fields.
	fields := mr.Header.Fields()
	for fields.Next() {
		text, err := fields.Text()
		if err != nil {
			text = ""
		}
		out.Headers[fields.Key()] = append(out.Headers[fields.Key()], text)
	}

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break // tolerate a truncated/invalid trailing part
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			body, _ := io.ReadAll(p.Body)
			ct, _, _ := h.ContentType()
			switch {
			case strings.HasPrefix(ct, "text/html"):
				out.HTML = string(body)
			case strings.HasPrefix(ct, "text/plain"):
				out.Text = string(body)
			default:
				if out.Text == "" {
					out.Text = string(body)
				}
			}
		case *mail.AttachmentHeader:
			body, _ := io.ReadAll(p.Body)
			filename, _ := h.Filename()
			ct, _, _ := h.ContentType()
			out.Files = append(out.Files, attachment{Filename: filename, ContentType: ct, Data: body})
		}
	}
	return out, nil
}

// displayContent returns the best body to show: HTML if present, else text.
func (p *parsedEmail) displayContent() string {
	if p.HTML != "" {
		return p.HTML
	}
	return p.Text
}
