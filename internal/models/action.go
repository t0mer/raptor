package models

import "time"

// Action is one step in a token's Custom Action chain. Steps run in `Position`
// order; `Parameters` is a type-specific JSON object; `Condition` optionally
// names another action whose outcome gates this one.
type Action struct {
	UUID       string         `json:"uuid"`
	TokenID    string         `json:"token_id"`
	Type       string         `json:"type"`
	Position   int            `json:"position"`
	Name       string         `json:"name"`
	Disabled   bool           `json:"disabled"`
	Parameters map[string]any `json:"parameters"`
	Queue      bool           `json:"queue"`
	Delay      int            `json:"delay"`
	Condition  string         `json:"condition,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// ActionRun records the output and error of one action executed for one request,
// powering the per-request action log and replay.
type ActionRun struct {
	ID         string    `json:"id"`
	RequestID  string    `json:"request_id"`
	ActionID   string    `json:"action_id"`
	ActionType string    `json:"action_type"`
	ActionName string    `json:"action_name"`
	Position   int       `json:"position"`
	Output     string    `json:"output"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"created_at"`
}
