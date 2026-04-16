# 2026-04-16-00 Overall Component UML

## 组件图

```mermaid
flowchart LR
  ai[AI 应用] --> mux[mcpmux]

  subgraph monorepo[MCP Monorepo]
    subgraph svc[services]
      vcenter[services/vcenter<br/>MCP Server]
      ces[services/ces<br/>MCP Server(占位)]
    end

    subgraph shared[modules]
      mcpkit[modules/mcpkit<br/>MCP HTTP/SSE + Auth + Config + Error]
    end

    vcenter --> mcpkit
    ces --> mcpkit
  end

  mux -->|HTTP/SSE| vcenter
  mux -.->|HTTP/SSE| ces

  vcenter -->|govmomi| vcapi[vCenter / vSphere]
  ces -.->|Huawei SDK| cesapi[Huawei Cloud CES]
```
