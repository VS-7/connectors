---
id: TASK-OMNI-02
type: task
title: Implementar Omni MCP Server (Go 1.23 / mark3labs/mcp-go)
status: completed
priority: must
rank: zzzzz
estimate_h: 19.5
tier: integration
component: Omni MCP Server
component_id: node-omni-mcp
dependencies:
  - TASK-CORE-01
requirements:
  - RF002
  - RF009
  - RNF001
  - RNF003
  - RNF005
use_cases:
  - CDU001
  - CDU006
source: component:node-omni-mcp
order: 14
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:40Z"
completed_at: "2026-10-19T01:39:40Z"
---

# TASK-OMNI-02 · Implementar Omni MCP Server (Go 1.23 / mark3labs/mcp-go)

## Critérios de aceite

- [ ] Rotas desconhecidas retornam 404 e erros upstream retornam 502 com corpo padronizado.
- [ ] Rate limiting e CORS configurados e cobertos por teste de integração.
