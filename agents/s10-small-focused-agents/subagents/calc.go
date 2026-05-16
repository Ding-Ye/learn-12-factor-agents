// Package subagents holds the small focused agents the orchestrator
// composes. Each sub-agent has its own thread + its own tiny loop. They
// don't know the orchestrator exists.
package subagents

import (
	"encoding/json"
	"fmt"
)

// CalcInput is the orchestrator's request to the CalcAgent: a list of
// {op, a, b} steps to compute sequentially, threading the result.
type CalcInput struct {
	Steps []CalcStep `json:"steps"`
}

type CalcStep struct {
	Op string  `json:"op"` // "add" | "multiply"
	A  float64 `json:"a"`
	B  float64 `json:"b"`
}

// CalcOutput carries the final scalar result and a small trace for
// debugging.
type CalcOutput struct {
	Result float64  `json:"result"`
	Trace  []string `json:"trace"`
}

// CalcAgent runs each step in order. The previous step's result
// replaces `A` of the next step (mimicking "then" composition). Returns
// an error on unknown op.
func CalcAgent(in CalcInput) (CalcOutput, error) {
	var current float64
	var trace []string
	for i, step := range in.Steps {
		a := step.A
		if i > 0 {
			a = current
		}
		switch step.Op {
		case "add":
			current = a + step.B
		case "multiply":
			current = a * step.B
		default:
			return CalcOutput{}, fmt.Errorf("unknown op %q at step %d", step.Op, i)
		}
		trace = append(trace, fmt.Sprintf("%s %v %v = %v", step.Op, a, step.B, current))
	}
	return CalcOutput{Result: current, Trace: trace}, nil
}

// MarshalCalcInput is a small convenience so the orchestrator's
// subagent_call event captures the input as JSON bytes (matching the
// JSON-throughout serialization story of the curriculum).
func MarshalCalcInput(in CalcInput) []byte {
	b, _ := json.Marshal(in)
	return b
}
