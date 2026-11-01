package renodash

// Item is an individual update controlled by a Dashboard checkbox.
type Item struct {
	Action   string    `json:"action"`
	Branch   string    `json:"branch"`
	Title    string    `json:"title"`
	Section  string    `json:"section"`
	Checked  bool      `json:"checked"`
	Research *Research `json:"research,omitempty"`
}

// BulkAction is a known Dashboard operation without a branch identifier.
type BulkAction struct {
	Action  string `json:"action"`
	Checked bool   `json:"checked"`
}

// Dashboard contains the operations and unrecognized controls in source order within each slice.
type Dashboard struct {
	Items       []Item       `json:"items"`
	BulkActions []BulkAction `json:"bulk_actions"`
	// UnknownControls contains unrecognized checkbox comment bodies from known
	// Renovate operation sections, trimmed and in input order, including duplicates.
	// Comments in unknown sections or code blocks are not collected.
	UnknownControls []string `json:"unknown_controls"`
}
