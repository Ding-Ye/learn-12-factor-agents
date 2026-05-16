package triggers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WebhookTrigger ingests a HumanLayer-style webhook:
//
//	{
//	  "event": {
//	    "spec":   { "state": { "thread_id": "abc..." } },
//	    "status": { "response": "approved" }
//	  }
//	}
//
// Maps to resuming an existing thread with a human_response.
type WebhookTrigger struct{}

func (WebhookTrigger) Source() string { return "webhook" }

type webhookBody struct {
	Event struct {
		Spec struct {
			State struct {
				ThreadID string `json:"thread_id"`
			} `json:"state"`
		} `json:"spec"`
		Status struct {
			Response string `json:"response"`
		} `json:"status"`
	} `json:"event"`
}

func (WebhookTrigger) Trigger(r *http.Request) (Outcome, error) {
	var b webhookBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		return Outcome{}, fmt.Errorf("decode webhook body: %w", err)
	}
	if b.Event.Spec.State.ThreadID == "" {
		return Outcome{}, fmt.Errorf("webhook body missing event.spec.state.thread_id")
	}
	return Outcome{
		ResumeThreadID: b.Event.Spec.State.ThreadID,
		HumanResponse:  b.Event.Status.Response,
		Raw: map[string]any{
			"thread_id": b.Event.Spec.State.ThreadID,
			"response":  b.Event.Status.Response,
		},
	}, nil
}
