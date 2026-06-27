package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/notify"
	"github.com/t0mer/raptor/internal/store"
)

// scheduleView masks the secret notify URL; only a boolean is returned.
type scheduleView struct {
	UUID         string     `json:"uuid"`
	TokenID      string     `json:"token_id,omitempty"`
	Name         string     `json:"name"`
	Cron         string     `json:"cron"`
	TargetURL    string     `json:"target_url"`
	Method       string     `json:"method"`
	Body         string     `json:"body"`
	RunActions   bool       `json:"run_actions"`
	ExpectStatus int        `json:"expect_status"`
	Keyword      string     `json:"keyword"`
	CheckSSL     bool       `json:"check_ssl"`
	SSLDays      int        `json:"ssl_days"`
	HasNotify    bool       `json:"has_notify"`
	Enabled      bool       `json:"enabled"`
	LastRun      *time.Time `json:"last_run,omitempty"`
	NextRun      *time.Time `json:"next_run,omitempty"`
	LastStatus   string     `json:"last_status"`
	LastMessage  string     `json:"last_message"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func toScheduleView(s *models.Schedule) scheduleView {
	return scheduleView{
		UUID: s.UUID, TokenID: s.TokenID, Name: s.Name, Cron: s.Cron, TargetURL: s.TargetURL,
		Method: s.Method, Body: s.Body, RunActions: s.RunActions, ExpectStatus: s.ExpectStatus,
		Keyword: s.Keyword, CheckSSL: s.CheckSSL, SSLDays: s.SSLDays, HasNotify: s.NotifyURL != "",
		Enabled: s.Enabled, LastRun: s.LastRun, NextRun: s.NextRun, LastStatus: s.LastStatus,
		LastMessage: s.LastMessage, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

type scheduleRequest struct {
	TokenID      *string `json:"token_id"`
	Name         *string `json:"name"`
	Cron         *string `json:"cron"`
	TargetURL    *string `json:"target_url"`
	Method       *string `json:"method"`
	Body         *string `json:"body"`
	RunActions   *bool   `json:"run_actions"`
	ExpectStatus *int    `json:"expect_status"`
	Keyword      *string `json:"keyword"`
	CheckSSL     *bool   `json:"check_ssl"`
	SSLDays      *int    `json:"ssl_days"`
	NotifyURL    *string `json:"notify_url"`
	Enabled      *bool   `json:"enabled"`
}

func (a *API) listSchedules(w http.ResponseWriter, r *http.Request) {
	list, err := a.store.ListSchedules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list schedules")
		return
	}
	views := make([]scheduleView, 0, len(list))
	for _, s := range list {
		views = append(views, toScheduleView(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": views})
}

func (a *API) createSchedule(w http.ResponseWriter, r *http.Request) {
	var body scheduleRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if body.Cron == nil || *body.Cron == "" {
		writeError(w, http.StatusBadRequest, "cron is required")
		return
	}
	if body.NotifyURL != nil && !notify.Valid(*body.NotifyURL) {
		writeError(w, http.StatusBadRequest, "invalid notify URL")
		return
	}
	sc := &models.Schedule{UUID: uuid.NewString(), Method: "GET", SSLDays: 14, Enabled: true}
	applyScheduleRequest(sc, &body)
	if err := a.store.CreateSchedule(r.Context(), sc); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create schedule")
		return
	}
	writeJSON(w, http.StatusCreated, toScheduleView(sc))
}

func (a *API) getSchedule(w http.ResponseWriter, r *http.Request) {
	sc, ok := a.loadSchedule(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toScheduleView(sc))
}

func (a *API) updateSchedule(w http.ResponseWriter, r *http.Request) {
	sc, ok := a.loadSchedule(w, r)
	if !ok {
		return
	}
	var body scheduleRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if body.NotifyURL != nil && !notify.Valid(*body.NotifyURL) {
		writeError(w, http.StatusBadRequest, "invalid notify URL")
		return
	}
	applyScheduleRequest(sc, &body)
	if err := a.store.UpdateSchedule(r.Context(), sc); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update schedule")
		return
	}
	writeJSON(w, http.StatusOK, toScheduleView(sc))
}

func (a *API) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "scheduleID")
	if err := a.store.DeleteSchedule(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "schedule not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete schedule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listScheduleRuns(w http.ResponseWriter, r *http.Request) {
	sc, ok := a.loadSchedule(w, r)
	if !ok {
		return
	}
	runs, err := a.store.ListScheduleRuns(r.Context(), sc.UUID, queryInt(r, "limit", 50))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	if runs == nil {
		runs = []*models.ScheduleRun{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": runs})
}

func (a *API) runScheduleNow(w http.ResponseWriter, r *http.Request) {
	sc, ok := a.loadSchedule(w, r)
	if !ok {
		return
	}
	run := a.schedules.Run(r.Context(), sc)
	writeJSON(w, http.StatusOK, run)
}

func (a *API) loadSchedule(w http.ResponseWriter, r *http.Request) (*models.Schedule, bool) {
	id := chi.URLParam(r, "scheduleID")
	sc, err := a.store.GetSchedule(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "schedule not found")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load schedule")
		return nil, false
	}
	return sc, true
}

func applyScheduleRequest(sc *models.Schedule, body *scheduleRequest) {
	if body.TokenID != nil {
		sc.TokenID = *body.TokenID
	}
	if body.Name != nil {
		sc.Name = *body.Name
	}
	if body.Cron != nil {
		sc.Cron = *body.Cron
	}
	if body.TargetURL != nil {
		sc.TargetURL = *body.TargetURL
	}
	if body.Method != nil {
		sc.Method = *body.Method
	}
	if body.Body != nil {
		sc.Body = *body.Body
	}
	if body.RunActions != nil {
		sc.RunActions = *body.RunActions
	}
	if body.ExpectStatus != nil {
		sc.ExpectStatus = *body.ExpectStatus
	}
	if body.Keyword != nil {
		sc.Keyword = *body.Keyword
	}
	if body.CheckSSL != nil {
		sc.CheckSSL = *body.CheckSSL
	}
	if body.SSLDays != nil {
		sc.SSLDays = *body.SSLDays
	}
	if body.NotifyURL != nil {
		sc.NotifyURL = *body.NotifyURL
	}
	if body.Enabled != nil {
		sc.Enabled = *body.Enabled
	}
}
