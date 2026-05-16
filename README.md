# learn-12-factor-agents

> 12 因子，12 章，Go 1.22 重写一遍 —— [humanlayer/12-factor-agents](https://github.com/humanlayer/12-factor-agents) 的渐进式学习仓库

[English](./README.en.md) · 中文（本文）

每一章是一个独立的 Go module，专注上游 12 factors 中的一个机制。代码 ≤ 1000 行，文档双语 + 上游源码注解版。s01 是最小 wire-format（NextStep + Provider），s12 是 stateless reducer，中间逐步加 prompt、thread、tools、loop、HTTP server、human-in-loop、error recovery、orchestrator、triggers。

教学法借鉴 [shareAI-lab/learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) 的六段式（Problem / Solution / How It Works / What Changed / Try It / Upstream Source Reading）。

## 课程目录

| # | slug | title | factor | 状态 |
|---|---|---|---|---|
| s01 | natural-language-to-tool-calls | 最小 agent 原语：自然语言到工具调用 | 1 | ✅ |
| s02 | [own-your-prompts](./docs/zh/s02-own-your-prompts.md) | 自己控制 prompt | 2 | ✅ |
| s03 | [own-your-context-window](./docs/zh/s03-own-your-context-window.md) | 自己控制上下文窗口 (Thread + Event) | 3 | ✅ |
| s04 | [tools-are-structured-outputs](./docs/zh/s04-tools-are-structured-outputs.md) | 工具即结构化输出 | 4 | ✅ |
| s05 | [unify-execution-state](./docs/zh/s05-unify-execution-state.md) | 统一执行状态与业务状态 | 5 | ✅ |
| s06 | launch-pause-resume | 启动 / 暂停 / 恢复 HTTP API | 6 | ⏳ |
| s07 | contact-humans-with-tools | 用工具调用方式联系人类 | 7 | ⏳ |
| s08 | own-your-control-flow | 自己控制流程 | 8 | ⏳ |
| s09 | compact-errors | 错误进上下文，模型自愈 | 9 | ⏳ |
| s10 | small-focused-agents | 小而专一的子 agent 编排 | 10 | ⏳ |
| s11 | trigger-from-anywhere | 任意触发源（Webhook / Slack） | 11 | ⏳ |
| s12 | stateless-reducer | 无状态 reducer (replay + fork) | 12 | ⏳ |
| s_full | integration | 端到端集成 · 12 因子贯通走查 | — | ⏳ |
| App. A | agents-are-software | 附录 A · 智能体即软件 | — | ⏳ |
| App. B | upstream-map | 附录 B · 上游源码导读地图 | — | ⏳ |

## 快速开始

```bash
git clone https://github.com/Ding-Ye/learn-12-factor-agents.git
cd learn-12-factor-agents

# 跑 s01
cd agents/s01-natural-language-to-tool-calls
go test ./...
go run . "hello"
# → intent=done_for_now message="Hello! How can I assist you today?"
```

每一章都自带 `README.md` + 双语 docs。建议按章节顺序读：

1. 先读 `docs/zh/sNN-*.md` 或 `docs/en/sNN-*.md`
2. 再看 `agents/sNN-*/` 下的源码
3. 想看上游对照：`upstream-readings/sNN-*.md`

## 为什么是 12 章？

上游的 [`content/`](https://github.com/humanlayer/12-factor-agents/tree/main/content) 目录正好是 12 个 factor markdown 文件，每个 factor 描述一种工程模式。我们让每章对应一个 factor，配 30-700 行 Go 代码 + 单元测试。

## 上游致谢

- [humanlayer/12-factor-agents](https://github.com/humanlayer/12-factor-agents) by Dex Horthy & contributors. Apache 2.0 (code) + CC-BY-SA 4.0 (content).
- [shareAI-lab/learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) 的六段式教学法启发。

## 许可

本仓库 Go 代码以 Apache License 2.0 发布。`docs/` 与 `upstream-readings/` 中引用自上游 `content/*.md` 的段落保留 [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/) 许可并附原文件路径 + 行号。
