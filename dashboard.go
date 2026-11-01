package renodash

// Item is an individual update controlled by a Dashboard checkbox.
type Item struct {
	Action  string
	Branch  string
	Title   string
	Section string
	Checked bool
}

// BulkAction is a known Dashboard operation without a branch identifier.
type BulkAction struct {
	Action  string
	Checked bool
}

// Dashboard contains the operations and unrecognized controls in source order within each slice.
type Dashboard struct {
	Items       []Item
	BulkActions []BulkAction
	// UnknownControls contains unrecognized checkbox comment bodies from known
	// Renovate operation sections, trimmed and in input order, including duplicates.
	// Comments in unknown sections or code blocks are not collected.
	UnknownControls []string
}
