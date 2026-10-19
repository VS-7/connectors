---
id: TASK-RF009-OMNI-2
type: task
title: Implementar ativação Seletiva e Modulação de Ferramentas em Omni HTTP Gateway (RF009)
status: completed
priority: must
rank: zzzzzzzzy
parent: ST-RF009
estimate_h: 1.5
tier: integration
component: Omni HTTP Gateway
component_id: node-omni-http
dependencies:
  - TASK-OMNI-01
requirements:
  - RF009
use_cases:
  - CDU006
source: slice:RF009:node-omni-http
order: 38
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:53Z"
completed_at: "2026-10-19T01:39:53Z"
---

# TASK-RF009-OMNI-2 · Implementar ativação Seletiva e Modulação de Ferramentas em Omni HTTP Gateway (RF009)

## Critérios de aceite

- [ ] RF009 atendido em Omni HTTP Gateway: Ativação Seletiva e Modulação de Ferramentas
- [ ] **Given** o comando `omni-mcp --enable=fathom` **When** o cliente executa `tools/list` **Then** apenas ferramentas com prefixo `fathom_` são listadas.
- [ ] **Given** uma chamada a `GET /v1/health` **When** o gateway foi iniciado com `--enable=slack,mattermost` **Then** o JSON de resposta exibe `enabled_connectors: ["slack", "mattermost"]`.
