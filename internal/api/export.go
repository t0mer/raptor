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
				req.UUID, req.Type, req.Method, req.IP, req.Hostname, req.UserAgent,
				req.URL, strconv.Itoa(req.Size), req.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				req.Content,
			})
		}
		cw.Flush()
		if len(reqs) < pageSize {
			break
		}
	}
}
