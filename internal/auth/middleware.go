package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

// Cookie names.
const (
	SessionCookie = "raptor_session" // login session
	OwnerCookie   = "raptor_owner"   // anonymous owner identity
)

// AnonPrefix marks an anonymous owner id so it can never collide with a user id.
const AnonPrefix = "anon-"

// ownerCookieMaxAge keeps anonymous identities for a year.
const ownerCookieMaxAge = 365 * 24 * 60 * 60

type ctxKey int

const (
	userKey ctxKey = iota
	ownerKey
)

// publicPaths are reachable without authentication so login, registration and
// first-run bootstrap work even when the API is gated.
var publicPaths = map[string]bool{
	"/api/v1/auth/login":     true,
	"/api/v1/auth/status":    true,
	"/api/v1/auth/bootstrap": true,
	"/api/v1/auth/register":  true,
}

// UserFromContext returns the authenticated (registered) user, if any.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userKey).(*models.User)
	return u, ok
}

// OwnerFromContext returns the request's owner id — the registered user's id, or
// the anonymous owner id (from the owner cookie). Empty only in private mode
// before authentication.
func OwnerFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ownerKey).(string)
	return s
}

func withUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func withOwner(ctx context.Context, owner string) context.Context {
	return context.WithValue(ctx, ownerKey, owner)
}

// Middleware authenticates the request and establishes its owner identity.
//
//   - A registered user (API key, session cookie or Basic Auth) owns by user id.
//   - Otherwise, unless requireAuth is set, an anonymous owner cookie is issued
//     and resolved, so every visitor gets an isolated identity without logging in.
//   - When requireAuth is on and the instance is bootstrapped, unauthenticated
//     access to non-public paths is rejected.
func (s *Service) Middleware(requireAuth, secureCookies bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := s.identify(r)

			owner := ""
			if user != nil {
				r = r.WithContext(withUser(r.Context(), user))
				owner = user.ID
			} else if !requireAuth {
				owner = ensureOwnerCookie(w, r, secureCookies)
			}
			if owner != "" {
				r = r.WithContext(withOwner(r.Context(), owner))
			}

			if publicPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			if requireAuth && user == nil && s.Bootstrapped(r.Context()) {
				unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ensureOwnerCookie returns the anonymous owner id from the request cookie,
// issuing and setting a fresh one when absent. The cookie is accepted only if it
// carries the anonymous prefix — a client-supplied value cannot impersonate a
// registered user's id (whose tokens it would otherwise be able to claim).
func ensureOwnerCookie(w http.ResponseWriter, r *http.Request, secure bool) string {
	if c, err := r.Cookie(OwnerCookie); err == nil && strings.HasPrefix(c.Value, AnonPrefix) {
		return c.Value
	}
	id := newAnonID()
	http.SetCookie(w, &http.Cookie{
		Name:     OwnerCookie,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   ownerCookieMaxAge,
		Expires:  time.Now().Add(ownerCookieMaxAge * time.Second),
	})
	return id
}

func newAnonID() string {
	tok, err := GenerateToken(24)
	if err != nil {
		// Extremely unlikely; fall back to a time-seeded value.
		tok = time.Now().UTC().Format("20060102150405.000000000")
	}
	return AnonPrefix + tok
}

// ClearOwnerCookie expires the anonymous owner cookie (after upgrade to an account).
func ClearOwnerCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     OwnerCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// RequireAdmin wraps handlers that only administrators may call.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok {
			unauthorized(w)
			return
		}
		if !u.IsAdmin() {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"admin role required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// identify resolves the request's user via API key, session cookie, or Basic
// Auth. Returns nil when unauthenticated.
func (s *Service) identify(r *http.Request) *models.User {
	ctx := r.Context()
	if key := strings.TrimSpace(r.Header.Get("Api-Key")); key != "" {
		if u, err := s.UserByAPIKey(ctx, key); err == nil {
			return u
		}
	}
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		if u, err := s.UserBySession(ctx, c.Value); err == nil {
			return u
		}
	}
	if email, pass, ok := r.BasicAuth(); ok {
		if u, err := s.Login(ctx, email, pass); err == nil {
			return u
		}
	}
	return nil
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"authentication required"}`))
}
