package subagents

import "fmt"

// SummaryInput is what the orchestrator hands to the SummaryAgent.
type SummaryInput struct {
	UserMessage string  `json:"user_message"`
	Result      float64 `json:"result"`
}

// SummaryAgent turns a numeric result into a short user-facing line.
// A real implementation would call an LLM here; ours uses a fixed
// template so tests stay deterministic.
func SummaryAgent(in SummaryInput) string {
	return fmt.Sprintf("Computed %v for: %s", in.Result, in.UserMessage)
}
