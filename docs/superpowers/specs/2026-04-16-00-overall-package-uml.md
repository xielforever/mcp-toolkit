# 2026-04-16-00 Overall Package UML

## 包图（go.work + 多 go.mod）

```mermaid
classDiagram
  class "go.work" as gowork <<workspace>>

  class "modules/mcpkit" as mcpkit <<module>>
  class "services/vcenter" as vcenter <<module>>
  class "services/ces" as ces <<module>>
  class "deploy/mcpmux" as deploymux <<package>>
  class "docs/superpowers/specs" as specs <<package>>

  gowork ..> mcpkit : use
  gowork ..> vcenter : use
  gowork ..> ces : use

  vcenter --> mcpkit : require
  ces --> mcpkit : require

  deploymux ..> vcenter : routes to
  deploymux ..> ces : routes to(未来)

  specs ..> mcpkit : design
  specs ..> vcenter : design
  specs ..> deploymux : design
```
