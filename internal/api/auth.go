package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/t0mer/raptor/internal/auth"
	"github.com/t0mer/raptor/internal/models"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authStatus reports whether the instance has been bootstrapped, whether auth is
// required, and who (if anyone) the caller is.
func (a *API) authStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"bootstrapped":       a.auth.Bootstrapped(r.Context()),
		"require_auth":       a.requireAuth,
		"allow_registration": a.allowRegistration,
	}
	if u, ok := auth.UserFromContext(r.Context()); ok {
		resp["authenticated"] = true
		resp["user"] = u
	} else {
		resp["authenticated"] = false
	}
	writeJSON(w, http.StatusOK, resp)
}

// bootstrap creates the first admin account when none exists, then logs them in.
func (a *API) bootstrap(w http.ResponseWriter, r *http.Request) {
	if a.auth.Bootstrapped(r.Context()) {
		writeError(w, http.StatusConflict, "already initialised")
		return
	}
	var c credentials
	if err := decodeJSON(w, r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if len(c.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	u, err := a.auth.CreateUser(r.Context(), c.Email, c.Password, models.RoleAdmin)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	a.completeLogin(w, r, u)
	writeJSON(w, http.StatusCreated, u)
}

// register creates a new account. The first account is always allowed (and made
// admin); subsequent self-registration requires AllowRegistration. On success
// the new account adopts the visitor's anonymous URLs and is logged in.
func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := decodeJSON(w, r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if len(c.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	bootstrapped := a.auth.Bootstrapped(r.Context())
	if bootstrapped && !a.allowRegistration {
		writeError(w, http.StatusForbidden, "registration is disabled")
		return
	}
	role := models.RoleUser
	if !bootstrapped {
		role = models.RoleAdmin // first account administers the instance
	}
	u, err := a.auth.CreateUser(r.Context(), c.Email, c.Password, role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	a.completeLogin(w, r, u)
	writeJSON(w, http.StatusCreated, u)
}

// login verifies credentials and starts a session.
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := decodeJSON(w, r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	u, err := a.auth.Login(r.Context(), c.Email, c.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	a.completeLogin(w, r, u)
	writeJSON(w, http.StatusOK, u)
}

// logout ends the current session and clears the cookie.
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.SessionCookie); err == nil && c.Value != "" {
		_ = a.auth.EndSession(r.Context(), c.Value)
	}
	http.SetCookie(w, a.sessionCookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}

// me returns the authenticated user.
func (a *API) me(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// completeLogin migrates any anonymous URLs onto the account, clears the
// anonymous owner cookie, and starts a session.
func (a *API) completeLogin(w http.ResponseWriter, r *http.Request, u *models.User) {
	if c, err := r.Cookie(auth.OwnerCookie); err == nil && strings.HasPrefix(c.Value, auth.AnonPrefix) {
		if _, err := a.store.ReassignTokens(r.Context(), c.Value, u.ID); err != nil {
			// Non-fatal: the user is still logged in, just without migration.
			writeError(w, http.StatusInternalServerError, "failed to migrate URLs")
			return
		}
		auth.ClearOwnerCookie(w, a.secureCookies)
	}
	sess, err := a.auth.StartSession(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start session")
		return
	}
	http.SetCookie(w, a.sessionCookie(sess.ID, int(time.Until(sess.ExpiresAt).Seconds())))
}

func (a *API) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

// currentUser returns the authenticated user or writes 401.
func (a *API) currentUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return nil, false
	}
	return u, true
}
