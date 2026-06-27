package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/store"
)

type groupRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (a *API) listGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := a.store.ListGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	if groups == nil {
		groups = []*models.Group{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": groups})
}

func (a *API) createGroup(w http.ResponseWriter, r *http.Request) {
	var body groupRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	name := ""
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
	}
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	g := &models.Group{ID: uuid.NewString(), Name: name}
	if body.Color != nil {
		g.Color = *body.Color
	}
	if err := a.store.CreateGroup(r.Context(), g); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create group")
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (a *API) updateGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "groupID")
	g, err := a.store.GetGroup(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load group")
		return
	}
	var body groupRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if body.Name != nil {
		g.Name = strings.TrimSpace(*body.Name)
	}
	if body.Color != nil {
		g.Color = *body.Color
	}
	if err := a.store.UpdateGroup(r.Context(), g); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update group")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (a *API) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "groupID")
	if err := a.store.DeleteGroup(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "group not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
