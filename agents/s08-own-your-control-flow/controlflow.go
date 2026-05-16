package main

// Action is the typed enum the control-flow function returns. Every loop
// iteration produces exactly one Action; RunAgent reacts based on the
// kind. New intents must map to one of these (and the test
// `TestControlFlow_AllIntentsCovered` enforces that).
type Action int

const (
	ActionInvalid  Action = iota
	ActionLoop            // execute the tool, append tool_response, continue
	ActionBreak           // append no tool_response, return cleanly (human pause)
	ActionFinish          // intent is done_for_now; loop terminates
	ActionEscalate        // unknown / un-routable intent; append error event, terminate
)

func (a Action) String() string {
	switch a {
	case ActionLoop:
		return "loop"
	case ActionBreak:
		return "break"
	case ActionFinish:
		return "finish"
	case ActionEscalate:
		return "escalate"
	default:
		return "invalid"
	}
}

// ControlFlow decides the next Action given the current thread and the
// LLM's most recent NextStep. The decision is data-driven (a switch on
// `next.Intent`); registry is consulted only to verify "do we have a
// tool for that intent?"
//
// Keeping ControlFlow pure (no I/O, no side effects) makes it trivial
// to test every branch.
func ControlFlow(_ *Thread, next NextStep, registry Registry) Action {
	switch next.Intent {
	case IntentDoneForNow:
		return ActionFinish
	case IntentRequestApproval, IntentRequestMoreInformation:
		return ActionBreak
	default:
		if _, ok := registry[next.Intent]; ok {
			return ActionLoop
		}
		return ActionEscalate
	}
}

// KnownIntents lists every intent ControlFlow has a non-Escalate branch
// for. Used by TestControlFlow_AllIntentsCovered to assert exhaustiveness
// against the closed set of constants in types.go.
func KnownIntents() []string {
	return []string{
		IntentDoneForNow,
		IntentRequestApproval,
		IntentRequestMoreInformation,
		IntentAdd,
		IntentMultiply,
	}
}
