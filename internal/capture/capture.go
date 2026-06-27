// Package capture records inbound HTTP requests against a token and writes the
// token's configured default response. It is the public-facing sink, so it
// enforces body-size limits, the per-token rate limit, request_limit and expiry.
package capture

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/metrics"
	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/store"
)

// DefaultMaxBodyBytes is the cap applied to captured request bodies.
const DefaultMaxBodyBytes int64 = 10 << 20 // 10 MiB

// Publisher receives newly captured requests for real-time fan-out (SSE). The
// no-op implementation is used until the SSE hub is wired in.
type Publisher interface {
	Publish(tokenID string, req *models.Request)
}

type nopPublisher struct{}

func (nopPublisher) Publish(string, *models.Request) {}

// Capturer records requests and renders default responses.
type Capturer struct {
	store        *store.Store
	baseURL      string
	maxBodyBytes int64
	globalLimit  int // config --max-requests; fallback when token.RequestLimit == 0
	limiter      *rateLimiter
	pub          Publisher
}

// Option configures a Capturer.
type Option func(*Capturer)

// WithPublisher sets the real-time publisher (SSE hub).
func WithPublisher(p Publisher) Option {
	return func(c *Capturer) {
		if p != nil {
			c.pub = p
		}
	}
}

// WithMaxBodyBytes overrides the captured-body size cap.
func WithMaxBodyBytes(n int64) Option {
	return func(c *Capturer) {
		if n > 0 {
			c.maxBodyBytes = n
		}
	}
}

// WithGlobalRequestLimit sets the fallback per-token stored-request cap.
func WithGlobalRequestLimit(n int) Option {
	return func(c *Capturer) { c.globalLimit = n }
}

// New constructs a Capturer.
func New(st *store.Store, baseURL string, opts ...Option) *Capturer {
	c := &Capturer{
		store:        st,
		baseURL:      baseURL,
		maxBodyBytes: DefaultMaxBodyBytes,
		limiter:      newRateLimiter(),
		pub:          nopPublisher{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Resolve looks up a token by its UUID, falling back to its alias. It returns
// store.ErrNotFound when neither matches.
func (c *Capturer) Resolve(ctx context.Context, identifier string) (*models.Token, error) {
	if identifier == "" {
		return nil, store.ErrNotFound
	}
	tok, err := c.store.GetToken(ctx, identifier)
	if err == nil {
		return tok, nil
	}
	if err != store.ErrNotFound {
		return nil, err
	}
	return c.store.GetTokenByAlias(ctx, identifier)
}

// Handle records the request against token and writes its default response.
// statusOverride, when non-nil, replaces the token's default response status
// (the /{tokenId}/{statusCode} form).
func (c *Capturer) Handle(w http.ResponseWriter, r *http.Request, token *models.Token, statusOverride *int) {
	now := time.Now().UTC()

	if IsExpired(token, now) {
		metrics.RequestsRejected.WithLabelValues("expired").Inc()
		http.Error(w, "this URL has expired", http.StatusGone)
		return
	}

	if !c.limiter.allow(token.UUID, token.Timeout) {
		metrics.RequestsRejected.WithLabelValues("rate_limited").Inc()
		w.Header().Set("Retry-After", "60")
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	req := c.buildRequest(r, token, now)

	limit := token.RequestLimit
	if limit == 0 {
		limit = c.globalLimit
	}
	if err := c.store.CreateRequest(r.Context(), req, limit); err != nil {
		http.Error(w, "failed to store request", http.StatusInternalServerError)
		return
	}
	metrics.RequestsCaptured.WithLabelValues(req.Type).Inc()
	c.pub.Publish(token.UUID, req)

	c.writeResponse(w, token, statusOverride)
}

func (c *Capturer) buildRequest(r *http.Request, token *models.Token, now time.Time) *models.Request {
	body, _ := io.ReadAll(io.LimitReader(r.Body, c.maxBodyBytes))

	headers := map[string][]string(r.Header.Clone())
	query := map[string][]string(r.URL.Query())

	return &models.Request{
		UUID:      uuid.NewString(),
		TokenID:   token.UUID,
		Type:      models.RequestTypeWeb,
		Method:    r.Method,
		IP:        clientIP(r),
		Hostname:  r.Host,
		UserAgent: r.UserAgent(),
		Content:   string(body),
		Query:     query,
		Headers:   headers,
		URL:       c.baseURL + r.URL.RequestURI(),
		Size:      len(body),
		Sorting:   now.UnixMilli(),
		CreatedAt: now,
	}
}

func (c *Capturer) writeResponse(w http.ResponseWriter, token *models.Token, statusOverride *int) {
	h := w.Header()
	if token.CORS {
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "*")
	}

	if token.Redirect != "" {
		http.Redirect(w, &http.Request{}, token.Redirect, http.StatusFound)
		return
	}

	ct := token.DefaultContentType
	if ct == "" {
		ct = "text/plain"
	}
	h.Set("Content-Type", ct)

	status := token.DefaultStatus
	if status == 0 {
		status = http.StatusOK
	}
	if statusOverride != nil {
		status = *statusOverride
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, token.DefaultContent)
}

// IsExpired reports whether a token's TTL (expiry seconds from creation) has
// elapsed. A zero expiry means the token never expires.
func IsExpired(token *models.Token, now time.Time) bool {
	if token.Expiry <= 0 {
		return false
	}
	return now.After(token.CreatedAt.Add(time.Duration(token.Expiry) * time.Second))
}

// clientIP returns the best-effort client IP, honouring a single
// X-Forwarded-For hop (Raptor is expected to run behind a reverse proxy).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
