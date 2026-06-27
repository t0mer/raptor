package models

// File is a stored attachment associated with a captured request (HTTP
// multipart upload, email attachment, or action-produced file). The blob is
// written under the data directory; Path is the on-disk location.
type File struct {
	ID          string `json:"id"`
	RequestID   string `json:"request_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Path        string `json:"-"`
}
