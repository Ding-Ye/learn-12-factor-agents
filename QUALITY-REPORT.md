# Quality report — learn-12-factor-agents

Generated 2026-05-16 after Phase H. All P0 checks passed; minor P1 issues are documented for follow-up but do not block ship.

## Summary

| Check | Result |
|---|---|
| All 12 chapter Go modules build + test | ✅ 12/12 PASS |
| No cross-session imports | ✅ 0 violations |
| Bilingual docs parity (zh/en `##` headings equal) | ✅ 0 mismatches |
| Six-section spine present in every chapter doc | ✅ 12/12 |
| Upstream file references resolve (file exists locally) | ✅ 12/12 |
| Upstream line-range citations within file bounds | ⚠️ 4 off-by-one (P1) |
| GitHub Actions CI runs | ✅ 20/20 success (11 docs + 9 go) |

## P0 issues
**None.**

## P1 issues (4 — off-by-one line citations)

Citations specify a `#L<a>-L<b>` range where `<b>` is one greater than the file's `wc -l`. The cited content is correct and present; only the closing line number is off by one. Cause: `wc -l` counts newlines, so files without a trailing newline report one less than the apparent line count.

| Citation | File | Cited | Actual lines |
|---|---|---|---|
| `01-agent.py#L1-L26` (s01 + s03 zh/en) | `.learn/upstream/workshops/2025-07-16/walkthrough/01-agent.py` | L26 | 25 |
| `03-agent.py#L14-L37` (s05 zh) | `.learn/upstream/workshops/2025-07-16/walkthrough/03-agent.py` | L37 | 36 |
| `09-state.ts#L1-L23` (s06 zh/en) | `.learn/upstream/workshops/2025-07-16/walkthrough/09-state.ts` | L23 | 22 |

The GitHub permalinks built from `upstream_sha = d20c728...` still resolve correctly because GitHub's `#L<a>-L<b>` clamps to the last line. Cosmetic only.

**Recommendation**: leave as-is or open a follow-up PR adjusting the four citations.

## P2 issues (minor)

- The 12-factor `s10-small-focused-agents/subagents/` and `s11-trigger-from-anywhere/triggers/` sub-packages report `[no test files]` in `go test`. Their behaviour is covered indirectly by the parent module's `orchestrator_test.go` and `server_test.go`. Tests in the sub-packages would be redundant.
- Web shell (`web/` Next.js viewer) is **not** included. CI doesn't reference it. Future enhancement.

## Files audited

```
agents/s01-natural-language-to-tool-calls/   ✅ 4 src + 1 test, 5 tests pass
agents/s02-own-your-prompts/                 ✅ 4 src + 1 test, 6 tests pass
agents/s03-own-your-context-window/          ✅ 5 src + 1 test, 6 tests pass
agents/s04-tools-are-structured-outputs/     ✅ 4 src + 1 test, 7 tests pass
agents/s05-unify-execution-state/            ✅ 6 src + 1 test, 6 tests pass
agents/s06-launch-pause-resume/              ✅ 7 src + 1 test, 6 tests pass (race-clean)
agents/s07-contact-humans-with-tools/        ✅ 7 src + 1 test, 5 tests pass
agents/s08-own-your-control-flow/            ✅ 6 src + 1 test, 7 tests pass
agents/s09-compact-errors/                   ✅ 6 src + 1 test, 6 tests pass
agents/s10-small-focused-agents/             ✅ 4 src + 2 sub-pkg + 1 test, 6 tests pass
agents/s11-trigger-from-anywhere/            ✅ 6 src + 3 trigger sub-pkg + 1 test, 5 tests pass
agents/s12-stateless-reducer/                ✅ 5 src + 1 test, 6 tests pass

docs/zh/  15 files (12 chapters + s_full + 2 appendices + multi-model)
docs/en/  15 files — full parity
upstream-readings/  12 annotation files
```

## Verdict

**Ship.** Repository is functionally complete, internally consistent, license-clean, and all critical paths verified.
