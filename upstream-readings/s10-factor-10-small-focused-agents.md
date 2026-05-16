# Upstream reading — Factor 10: Small, focused agents

> Source: humanlayer/12-factor-agents @ d20c728.

## Source: `content/factor-10-small-focused-agents.md` (full)

Upstream factor-10 is markdown-only; there is no reference implementation
in `workshops/`. The argument:

> Most successful agents in production keep individual agent loops
> between 3-20 steps. Beyond that, two failure modes show up: context
> bloat and mixed concerns. The fix: compose small agents. Each one
> does one thing. The outer orchestrator (just normal code!) decides
> which agent runs when, and threads the data between them.
>
> _(content/factor-10-small-focused-agents.md, lines 1-41; CC-BY-SA 4.0)_

## Mapping

| Upstream concept | Our Go translation | File |
|---|---|---|
| Sub-agent ("small focused") | Function in `subagents/*.go` | subagents/calc.go, subagents/summary.go |
| Orchestrator (regular code) | `Orchestrate` function | orchestrator.go |
| Sub-agent thread isolation | Sub-agents return values, don't touch the orchestrator thread | (by construction) |
| Orchestrator thread records boundaries | `subagent_call` / `subagent_done` events | events.go |

## Why we don't make sub-agents use the same `Provider`/`Tool` machinery

Real production sub-agents WOULD use a Provider — they're agents! For
teaching purposes our `CalcAgent` is a pure function so:

1. The orchestrator pattern is visible without LLM noise.
2. Tests are deterministic without a stub provider per sub-agent.
3. The handoff data structures (`CalcInput`, `CalcOutput`) are easier
   to read than a thread.

Extension exercise (Appendix B #5): make `CalcAgent` use a real
Provider + scripted sequence.

## Reading map

- s10 → s11: triggers (Slack / webhook) become external entry points
  that produce threads — symmetric to how the orchestrator produces
  sub-agent inputs.
- s10 → s12: the orchestrator becomes a special case of a Reduce over
  sub-agent events.
