# 2026-04-16-00 Overall Deployment UML

## 部署图（docker-compose 视角）

```mermaid
flowchart TB
  subgraph docker[docker-compose]
    subgraph net[mcp-network]
      mux[mcpmux<br/>container]
      vcenter[vcenter-mcp-server<br/>container :8080]
      ces[ces-mcp-server<br/>container :8080(未来)]
    end
  end

  ai[AI 应用] -->|MCP client| mux

  mux -->|GET /sse<br/>Authorization: Bearer| vcenter
  mux -->|POST /messages<br/>Authorization: Bearer| vcenter
  mux -->|GET /readyz| vcenter

  mux -.->|GET /sse<br/>Authorization: Bearer| ces
  mux -.->|POST /messages<br/>Authorization: Bearer| ces

  vcenter --> vc[vCenter / vSphere]
  ces -.-> hces[Huawei Cloud CES]
```
