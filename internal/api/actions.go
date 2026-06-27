package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/store"
)

type actionRequest struct {
	Type       *string        `json:"type"`
	Position   *int           `json:"position"`
	Name       *string        `json:"name"`
	Disabled   *bool          `json:"disabled"`
	Parameters map[string]any `json:"parameters"`
	Queue      *bool          `json:"queue"`
	Delay      *int           `json:"delay"`
	Condition  *string        `json:"condition"`
}

func (a *API) listActions(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	acts, err := a.store.ListActions(r.Context(), tok.UUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list actions")
		return
	}
	if acts == nil {
		acts = []*models.Action{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": acts})
}

func (a *API) createAction(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	var body actionRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if body.Type == nil || *body.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	act := &models.Action{
		UUID:       uuid.NewString(),
		TokenID:    tok.UUID,
		Type:       *body.Type,
		Parameters: map[string]any{},
	}
	applyActionRequest(act, &body)
	if err := a.store.CreateAction(r.Context(), act); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create action")
		return
	}
	writeJSON(w, http.StatusCreated, act)
}

func (a *API) updateAction(w http.ResponseWriter, r *http.Request) {
	act, ok := a.loadAction(w, r)
	if !ok {
		return
	}
	var body actionRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	applyActionRequest(act, &body)
	if err := a.store.UpdateAction(r.Context(), act); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update action")
		return
	}
	writeJSON(w, http.StatusOK, act)
}

func (a *API) deleteAction(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.loadAction(w, r); !ok {
		return
	}
	if err := a.store.DeleteAction(r.Context(), chi.URLParam(r, "actionID")); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete action")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testAction runs a single (unsaved) action definition against the token's most
// recent request — or a synthetic empty one — and returns its output and the
// resulting response/variables, without persisting anything.
func (a *API) testAction(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	var body actionRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if body.Type == nil || *body.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	def := &models.Action{UUID: "test", TokenID: tok.UUID, Type: *body.Type, Parameters: map[string]any{}}
	applyActionRequest(def, &body)

	sample, err := a.store.LatestRequest(r.Context(), tok.UUID)
	if errors.Is(err, store.ErrNotFound) || sample == nil {
		sample = &models.Request{TokenID: tok.UUID, Type: models.RequestTypeWeb}
	}

	ec, results := a.actions.ExecuteDefs(r.Context(), []*models.Action{def}, sample, tok)
	resErr := ""
	if len(results) > 0 && results[0].Err != nil {
		resErr = results[0].Err.Error()
	}
	out := ""
	if len(results) > 0 {
		out = results[0].Output
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"output":    out,
		"error":     resErr,
		"variables": ec.Vars,
		"response": map[string]any{
			"status":       ec.Response.Status,
			"content":      ec.Response.Content,
			"content_type": ec.Response.ContentType,
		},
		"dont_save": ec.DontSave,
		"stopped":   ec.Stopped,
	})
}

// executeChain replays the token's whole action chain against a stored request,
// persisting fresh run logs, and returns the run list.
func (a *API) executeChain(w http.ResponseWriter, r *http.Request) {
	req, ok := a.loadRequest(w, r)
	if !ok {
		return
	}
	tok, err := a.store.GetToken(r.Context(), req.TokenID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load token")
		return
	}
	_, results, err := a.actions.Execute(r.Context(), req, tok)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to execute actions")
		return
	}
	if err := a.actions.SaveRuns(r.Context(), req.UUID, results); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save action runs")
		return
	}
	runs, _ := a.store.ListActionRuns(r.Context(), req.UUID)
	writeJSON(w, http.StatusOK, map[string]any{"data": runs})
}

func (a *API) listActionRuns(w http.ResponseWriter, r *http.Request) {
	req, ok := a.loadRequest(w, r)
	if !ok {
		return
	}
	runs, err := a.store.ListActionRuns(r.Context(), req.UUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list action runs")
		return
	}
	if runs == nil {
		runs = []*models.ActionRun{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": runs})
}

func (a *API) loadAction(w http.ResponseWriter, r *http.Request) (*models.Action, bool) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return nil, false
	}
	id := chi.URLParam(r, "actionID")
	act, err := a.store.GetAction(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) || (act != nil && act.TokenID != tok.UUID) {
		writeError(w, http.StatusNotFound, "action not found")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load action")
		return nil, false
	}
	return act, true
}

func applyActionRequest(act *models.Action, body *actionRequest) {
	if body.Type != nil {
		act.Type = *body.Type
	}
	if body.Position != nil {
		act.Position = *body.Position
	}
	if body.Name != nil {
		act.Name = *body.Name
	}
	if body.Disabled != nil {
		act.Disabled = *body.Disabled
	}
	if body.Parameters != nil {
		act.Parameters = body.Parameters
	}
	if body.Queue != nil {
		act.Queue = *body.Queue
	}
	if body.Delay != nil {
		act.Delay = *body.Delay
	}
	if body.Condition != nil {
		act.Condition = *body.Condition
	}
}
