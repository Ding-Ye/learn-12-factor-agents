# learn-12-factor-agents

> 12 factors, 12 chapters, rewritten in Go 1.22 — a progressive learning companion for [humanlayer/12-factor-agents](https://github.com/humanlayer/12-factor-agents)

English (this file) · [中文](./README.md)

Each chapter is a standalone Go module focused on one of the upstream 12 factors. Code stays under ~1000 lines per chapter; docs are bilingual, paired with annotated upstream source readings. s01 nails the minimum wire format (`NextStep` + `Provider`); s12 lands the stateless reducer. In between, we add prompt templating, thread/event log, structured tools, the agent loop, an HTTP server, human-in-loop, error recovery, orchestrator, and pluggable triggers.

Pedagogy borrowed from [shareAI-lab/learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) (six-section spine: Problem / Solution / How It Works / What Changed / Try It / Upstream Source Reading).

## Curriculum

| # | slug | title | factor | status |
|---|---|---|---|---|
| s01 | natural-language-to-tool-calls | Minimum agent primitive — NL → tool calls | 1 | ✅ |
| s02 | [own-your-prompts](./docs/en/s02-own-your-prompts.md) | Own your prompts | 2 | ✅ |
| s03 | [own-your-context-window](./docs/en/s03-own-your-context-window.md) | Own your context window (Thread + Event) | 3 | ✅ |
| s04 | [tools-are-structured-outputs](./docs/en/s04-tools-are-structured-outputs.md) | Tools are structured outputs | 4 | ✅ |
| s05 | unify-execution-state | Unify execution state | 5 | ⏳ |
| s06 | launch-pause-resume | Launch / pause / resume HTTP API | 6 | ⏳ |
| s07 | contact-humans-with-tools | Contact humans with tools | 7 | ⏳ |
| s08 | own-your-control-flow | Own your control flow | 8 | ⏳ |
| s09 | compact-errors | Compact errors into context | 9 | ⏳ |
| s10 | small-focused-agents | Small, focused agents | 10 | ⏳ |
| s11 | trigger-from-anywhere | Trigger from anywhere (webhook / Slack) | 11 | ⏳ |
| s12 | stateless-reducer | Stateless reducer (replay + fork) | 12 | ⏳ |
| s_full | integration | End-to-end integration | — | ⏳ |
| App. A | agents-are-software | Appendix A · Agents are mostly software | — | ⏳ |
| App. B | upstream-map | Appendix B · Upstream reading map | — | ⏳ |

## Quickstart

```bash
git clone https://github.com/Ding-Ye/learn-12-factor-agents.git
cd learn-12-factor-agents

# Run s01
cd agents/s01-natural-language-to-tool-calls
go test ./...
go run . "hello"
# → intent=done_for_now message="Hello! How can I assist you today?"
```

Each chapter ships a `README.md` plus bilingual docs. The suggested reading order:

1. Read `docs/en/sNN-*.md` (or `docs/zh/sNN-*.md`).
2. Walk the source under `agents/sNN-*/`.
3. Compare against upstream via `upstream-readings/sNN-*.md`.

## Why 12 chapters?

Upstream's [`content/`](https://github.com/humanlayer/12-factor-agents/tree/main/content) is exactly twelve `factor-*.md` files, each describing one engineering pattern. We make every chapter mirror one factor with 30–700 lines of Go + unit tests.

## Upstream credits

- [humanlayer/12-factor-agents](https://github.com/humanlayer/12-factor-agents) by Dex Horthy & contributors. Apache 2.0 (code) + CC-BY-SA 4.0 (content).
- [shareAI-lab/learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) inspired the six-section pedagogy.

## License

Go code under Apache License 2.0. Quoted passages in `docs/` and `upstream-readings/` from upstream `content/*.md` remain under [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/) and are attributed with file path + line numbers at the point of use.
