---
id: TASK-RF009-OMNI
type: task
title: Implementar ativação Seletiva e Modulação de Ferramentas em Omni MCP Server (RF009)
status: completed
priority: must
rank: zzzzzzzzw
parent: ST-RF009
estimate_h: 1.5
tier: integration
component: Omni MCP Server
component_id: node-omni-mcp
dependencies:
  - TASK-OMNI-02
requirements:
  - RF009
use_cases:
  - CDU006
source: slice:RF009:node-omni-mcp
order: 37
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF009-OMNI · Implementar ativação Seletiva e Modulação de Ferramentas em Omni MCP Server (RF009)

## Critérios de aceite

- [ ] RF009 atendido em Omni MCP Server: Ativação Seletiva e Modulação de Ferramentas
- [ ] **Given** o comando `omni-mcp --enable=fathom` **When** o cliente executa `tools/list` **Then** apenas ferramentas com prefixo `fathom_` são listadas.
- [ ] **Given** uma chamada a `GET /v1/health` **When** o gateway foi iniciado com `--enable=slack,mattermost` **Then** o JSON de resposta exibe `enabled_connectors: ["slack", "mattermost"]`.
