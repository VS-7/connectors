---
id: TASK-RF003-OMNI
type: task
title: Implementar gateway REST de OpenAI Function Calling em Omni HTTP Gateway (RF003)
status: completed
priority: must
rank: zzzzzy
parent: ST-RF003
estimate_h: 3
tier: integration
component: Omni HTTP Gateway
component_id: node-omni-http
dependencies:
  - TASK-OMNI-01
requirements:
  - RF003
use_cases:
  - CDU002
source: slice:RF003:node-omni-http
order: 20
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF003-OMNI · Implementar gateway REST de OpenAI Function Calling em Omni HTTP Gateway (RF003)

## Critérios de aceite

- [ ] RF003 atendido em Omni HTTP Gateway: Gateway REST de OpenAI Function Calling
- [ ] **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- [ ] **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
