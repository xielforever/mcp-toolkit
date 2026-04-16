# 2026-04-16-06 CES Service Skeleton Design

## 目标

在一期不实现华为云 CES 具体能力的前提下，预先在 Monorepo 中建立 `services/ces` 模块脚手架，使其与 `vcenter` 服务共享同一套 `modules/mcpkit` 基础设施，并为后续接入 CES 指标查询与告警能力预留合理边界。

## 模块布局（services/ces）

建议目录：

```
services/ces/
├── go.mod
├── cmd/
│   └── ces-mcp-server/
│       └── main.go
└── internal/
    ├── config/
    ├── ces/                   # Huawei Cloud CES client 封装（后续填充）
    └── tools/                 # MCP tools（后续填充）
```

## 配置占位（环境变量）

鉴权与服务端通用配置沿用 `mcpkit`：

- `HTTP_LISTEN_ADDR`
- `MCP_HTTP_BASE_PATH`
- `MCP_SERVICE_TOKEN`

华为云认证与区域等后续再确定，预留命名（不在一期强制启用）：

- `HUAWEI_CLOUD_REGION`
- `HUAWEI_CLOUD_AK`
- `HUAWEI_CLOUD_SK`
- `HUAWEI_CLOUD_PROJECT_ID`（如需要）

## Tool 命名预留（不实现）

后续建议按以下前缀命名：

- `ces.query_metric`
- `ces.list_metrics`
- `ces.list_alarms`

具体字段与语义在 CES 功能设计阶段再补充。
