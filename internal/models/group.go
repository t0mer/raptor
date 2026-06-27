package models

import "time"

// Group organises capture tokens (URLs) under a named, optionally colour-coded
// bucket. A token references its group via Token.GroupID.
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
