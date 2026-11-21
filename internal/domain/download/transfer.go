package download

import "time"

// Transfer is the durable checkpoint for an explicitly selected resource.
// It never contains catalogue metadata or credentials.
type Transfer struct {
	URI string `json:"uri"`
	Stage string `json:"stage"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Files []TransferFile `json:"files,omitempty"`
	Directories []string `json:"directories,omitempty"`
}

type TransferFile struct {
	ID string `json:"id"`
	Name string `json:"name"`
	OriginalName string `json:"original_name,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}
