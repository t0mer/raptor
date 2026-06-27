package email

import (
	"bytes"
	"net"
	"strings"

	"blitiri.com.ar/go/spf"
	"github.com/emersion/go-msgauth/dkim"
	"github.com/emersion/go-msgauth/dmarc"
)

// Checker evaluates message authentication and returns a result map suitable for
// storage in Request.Checks. It must be best-effort and never panic.
type Checker func(raw []byte, ip net.IP, helo, mailFrom string) map[string]any

// realChecks runs DKIM, SPF and DMARC evaluation. SPF and DMARC perform DNS
// lookups; any transient failure is reported rather than fatal.
func realChecks(raw []byte, ip net.IP, helo, mailFrom string) map[string]any {
	out := map[string]any{
		"dkim":  dkimResult(raw),
		"spf":   spfResult(ip, helo, mailFrom),
		"dmarc": dmarcResult(mailFrom),
	}
	return out
}

func dkimResult(raw []byte) string {
	verifications, err := dkim.Verify(bytes.NewReader(raw))
	if err != nil || len(verifications) == 0 {
		return "none"
	}
	for _, v := range verifications {
		if v.Err == nil {
			return "pass"
		}
	}
	return "fail"
}

func spfResult(ip net.IP, helo, mailFrom string) string {
	if ip == nil || mailFrom == "" {
		return "none"
	}
	res, _ := spf.CheckHostWithSender(ip, helo, mailFrom)
	if res == "" {
		return "none"
	}
	return string(res)
}

func dmarcResult(mailFrom string) string {
	domain := domainOf(mailFrom)
	if domain == "" {
		return "none"
	}
	rec, err := dmarc.Lookup(domain)
	if err != nil {
		return "none"
	}
	if rec.Policy == "" {
		return "none"
	}
	return "policy:" + string(rec.Policy)
}

func domainOf(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		return strings.ToLower(addr[i+1:])
	}
	return ""
}
