---
id: TASK-RF003-CORE
type: task
title: Implementar gateway REST de OpenAI Function Calling em Core Connectors Hub (RF003)
status: completed
priority: must
rank: zzzzzz
parent: ST-RF003
estimate_h: 3
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF003
use_cases:
  - CDU002
source: slice:RF003:node-core-hub
order: 21
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF003-CORE · Implementar gateway REST de OpenAI Function Calling em Core Connectors Hub (RF003)

## Critérios de aceite

- [ ] RF003 atendido em Core Connectors Hub: Gateway REST de OpenAI Function Calling
- [ ] **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- [ ] **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
