package triggers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackTrigger ingests a stripped-down Slack event payload:
//
//	{ "event": { "type": "message", "text": "...", "channel": "C123" } }
//
// Real Slack delivers more fields and requires HMAC signature
// verification. Both are extension exercises (Appendix B).
type SlackTrigger struct{}

func (SlackTrigger) Source() string { return "slack" }

type slackBody struct {
	Event struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Channel string `json:"channel"`
	} `json:"event"`
}

func (SlackTrigger) Trigger(r *http.Request) (Outcome, error) {
	var b slackBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		return Outcome{}, fmt.Errorf("decode slack body: %w", err)
	}
	if b.Event.Type != "message" || b.Event.Text == "" {
		return Outcome{}, fmt.Errorf("slack body is not a message event (got %q)", b.Event.Type)
	}
	return Outcome{
		FreshUserInput: b.Event.Text,
		Raw: map[string]any{
			"channel": b.Event.Channel,
			"text":    b.Event.Text,
		},
	}, nil
}
