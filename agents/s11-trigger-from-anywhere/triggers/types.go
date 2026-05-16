// Package triggers defines the Trigger contract and ships two
// implementations (webhook + slack). Trigger turns an HTTP request from
// an external source into a domain object the agent can run on:
// either a fresh seed for a new thread, or a follow-up event for an
// existing thread.
package triggers

import "net/http"

// Outcome is the union returned by Trigger.Trigger.
type Outcome struct {
	// FreshUserInput, if non-empty, indicates a new thread should be
	// spawned with this user_input as the seed.
	FreshUserInput string

	// ResumeThreadID + HumanResponse, if both non-empty, indicate an
	// existing thread should be resumed with this human_response event.
	ResumeThreadID string
	HumanResponse  string

	// Raw captures the trigger payload for audit. Always populated.
	Raw map[string]any
}

// IsFresh is true when the trigger spawns a new thread.
func (o Outcome) IsFresh() bool { return o.FreshUserInput != "" && o.ResumeThreadID == "" }

// IsResume is true when the trigger resumes an existing thread.
func (o Outcome) IsResume() bool { return o.ResumeThreadID != "" }

// Trigger is the contract every external source implements. The Source
// method returns a human-readable name (used in trigger events) and
// Trigger() parses the request.
type Trigger interface {
	Source() string
	Trigger(r *http.Request) (Outcome, error)
}
