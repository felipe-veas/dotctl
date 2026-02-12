package types

// StatusResponse is the JSON structure returned by `dotctl status --json`.
// This is the contract consumed by both macOS and Linux tray apps.
type StatusResponse struct {
	Profile  string        `json:"profile"`
	OS       string        `json:"os"`
	Arch     string        `json:"arch"`
	Repo     RepoStatus    `json:"repo"`
	Symlinks SymlinkStatus `json:"symlinks"`
	Auth     AuthStatus    `json:"auth"`
	Warnings []string      `json:"warnings,omitempty"`
	Errors   []string      `json:"errors"`
}

type RepoStatus struct {
	Name       string `json:"name,omitempty"`
	URL        string `json:"url"`
	Status     string `json:"status"`
	Branch     string `json:"branch,omitempty"`
	LastCommit string `json:"last_commit,omitempty"`
	LastSync   string `json:"last_sync,omitempty"`
}

type SymlinkStatus struct {
	Total   int             `json:"total"`
	OK      int             `json:"ok"`
	Broken  int             `json:"broken"`
	Drift   int             `json:"drift"`
	Details []SymlinkDetail `json:"details,omitempty"`
}

type SymlinkDetail struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Status string `json:"status"` // "ok", "broken", "drift"
	Error  string `json:"error,omitempty"`
}

type AuthStatus struct {
	Method string `json:"method"`
	User   string `json:"user,omitempty"`
	OK     bool   `json:"ok"`
}
