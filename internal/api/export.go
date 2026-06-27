package api

import (
	"encoding/csv"
	"net/http"
	"strconv"
)

// exportCSV streams all of a token's requests as a CSV download.
func (a *API) exportCSV(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="requests.csv"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{
		"uuid", "type", "method", "ip", "hostname", "user_agent",
		"url", "size", "created_at", "content",
	})

	const pageSize = 100
	for offset := 0; ; offset += pageSize {
		reqs, err := a.store.ListRequests(r.Context(), tok.UUID, pageSize, offset)
		if err != nil {
			// Headers already sent; abort the stream.
			return
		}
		if len(reqs) == 0 {
			break
		}
		for _, req := range reqs {
			_ = cw.Write([]string{
				csvSafe(req.UUID), csvSafe(req.Type), csvSafe(req.Method), csvSafe(req.IP),
				csvSafe(req.Hostname), csvSafe(req.UserAgent), csvSafe(req.URL),
				strconv.Itoa(req.Size), req.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				csvSafe(req.Content),
			})
		}
		cw.Flush()
		if len(reqs) < pageSize {
			break
		}
	}
}

// csvSafe neutralises spreadsheet formula injection. Captured request data is
// fully attacker-controlled, so any cell beginning with a formula trigger
// (= + - @, or a leading tab/CR) is prefixed with a single quote, which Excel
// and Google Sheets treat as a literal-text marker.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}
