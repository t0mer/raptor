package api

import (
	"time"

	"github.com/t0mer/raptor/internal/models"
)

// tokenView is the API representation of a token. The basic-auth password is
// never returned; only a boolean indicates whether one is set.
type tokenView struct {
	UUID               string     `json:"uuid"`
	Alias              string     `json:"alias,omitempty"`
	URL                string     `json:"url"`
	DefaultStatus      int        `json:"default_status"`
	DefaultContent     string     `json:"default_content"`
	DefaultContentType string     `json:"default_content_type"`
	Timeout            int        `json:"timeout"`
	CORS               bool       `json:"cors"`
	Expiry             int        `json:"expiry"`
	Actions            bool       `json:"actions"`
	RequestLimit       int        `json:"request_limit"`
	Description        string     `json:"description"`
	Listen             int        `json:"listen"`
	Redirect           string     `json:"redirect"`
	GroupID            string     `json:"group_id,omitempty"`
	Premium            bool       `json:"premium"`
	HasPassword        bool       `json:"has_password"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LatestRequestAt    *time.Time `json:"latest_request_at,omitempty"`
}

func (a *API) tokenView(t *models.Token) tokenView {
	return tokenView{
		UUID:               t.UUID,
		Alias:              t.Alias,
		URL:                a.captureURL(t),
		DefaultStatus:      t.DefaultStatus,
		DefaultContent:     t.DefaultContent,
		DefaultContentType: t.DefaultContentType,
		Timeout:            t.Timeout,
		CORS:               t.CORS,
		Expiry:             t.Expiry,
		Actions:            t.Actions,
		RequestLimit:       t.RequestLimit,
		Description:        t.Description,
		Listen:             t.Listen,
		Redirect:           t.Redirect,
		GroupID:            t.GroupID,
		Premium:            t.Premium,
		HasPassword:        t.Password != "",
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
		LatestRequestAt:    t.LatestRequestAt,
	}
}

// captureURL builds the copyable capture URL for a token.
func (a *API) captureURL(t *models.Token) string {
	id := t.UUID
	if t.Alias != "" {
		id = t.Alias
	}
	return a.baseURL + "/" + id
}
