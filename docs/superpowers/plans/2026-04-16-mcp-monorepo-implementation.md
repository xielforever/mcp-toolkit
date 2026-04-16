# MCP Monorepo Implementation Plan (Index)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于 6 份 specs（一份一份实现计划）完成 Go Monorepo（go.work + 多 go.mod）、mcpkit、vcenter MCP 服务、CES 占位服务与 mcpmux compose 接入示例。

**Architecture:** `go.work` 聚合多个 module；`modules/mcpkit` 提供通用 MCP HTTP/SSE + 鉴权能力；`services/*` 以独立 module 方式复用 mcpkit；`deploy/mcpmux` 提供集成示例。

---

## Specs（输入）

- `docs/superpowers/specs/2026-04-16-01-monorepo-workspace-design.md`
- `docs/superpowers/specs/2026-04-16-02-mcpkit-http-sse-design.md`
- `docs/superpowers/specs/2026-04-16-03-authn-authz-design.md`
- `docs/superpowers/specs/2026-04-16-04-vcenter-service-design.md`
- `docs/superpowers/specs/2026-04-16-05-mcpmux-compose-integration-design.md`
- `docs/superpowers/specs/2026-04-16-06-ces-service-skeleton-design.md`

## Implementation Plans（输出，一对一）

| Spec | Implementation Plan |
|---|---|
| 01 Monorepo Workspace | `docs/superpowers/plans/2026-04-16-01-monorepo-workspace-implementation.md` |
| 02 MCPKit HTTP/SSE | `docs/superpowers/plans/2026-04-16-02-mcpkit-http-sse-implementation.md` |
| 03 AuthN/AuthZ | `docs/superpowers/plans/2026-04-16-03-authn-authz-implementation.md` |
| 04 vCenter Service | `docs/superpowers/plans/2026-04-16-04-vcenter-service-implementation.md` |
| 05 mcpmux Compose Integration | `docs/superpowers/plans/2026-04-16-05-mcpmux-compose-integration-implementation.md` |
| 06 CES Service Skeleton | `docs/superpowers/plans/2026-04-16-06-ces-service-skeleton-implementation.md` |

## 推荐执行顺序（含依赖）

- 1) Spec 01：建立 workspace 与 modules
- 2) Spec 02：实现 mcpkit（httpkit/mcp/mcphttp）
- 3) Spec 03：实现 Bearer 鉴权并在服务侧挂载
- 4) Spec 04：实现 vCenter 服务（govmomi + tools）
- 5) Spec 06：实现 CES 占位服务
- 6) Spec 05：补齐 mcpmux compose 集成示例

