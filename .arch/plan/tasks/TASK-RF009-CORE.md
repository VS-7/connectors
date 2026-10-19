---
id: TASK-RF009-CORE
type: task
title: Implementar ativação Seletiva e Modulação de Ferramentas em Core Connectors Hub (RF009)
status: completed
priority: must
rank: zzzzzzzzz
parent: ST-RF009
estimate_h: 1.5
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF009
use_cases:
  - CDU006
source: slice:RF009:node-core-hub
order: 39
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:53Z"
completed_at: "2026-10-19T01:39:53Z"
---

# TASK-RF009-CORE · Implementar ativação Seletiva e Modulação de Ferramentas em Core Connectors Hub (RF009)

## Critérios de aceite

- [ ] RF009 atendido em Core Connectors Hub: Ativação Seletiva e Modulação de Ferramentas
- [ ] **Given** o comando `omni-mcp --enable=fathom` **When** o cliente executa `tools/list` **Then** apenas ferramentas com prefixo `fathom_` são listadas.
- [ ] **Given** uma chamada a `GET /v1/health` **When** o gateway foi iniciado com `--enable=slack,mattermost` **Then** o JSON de resposta exibe `enabled_connectors: ["slack", "mattermost"]`.
