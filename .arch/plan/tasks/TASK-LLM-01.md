---
id: TASK-LLM-01
type: task
title: Implementar LLM & Agent Clients (Claude / Antigravity / Codex / Spark)
status: completed
rank: zzzzzV
estimate_h: 39.5
tier: frontend
component: LLM & Agent Clients
component_id: node-llm-clients
dependencies:
  - TASK-OMNI-01
  - TASK-OMNI-02
use_cases:
  - CDU001
  - CDU002
endpoints:
  - GET /v1/health
  - GET /v1/tools
  - POST /v1/tools/execute
source: component:node-llm-clients
order: 15
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:40:14Z"
completed_at: "2026-10-19T01:40:14Z"
---

# TASK-LLM-01 · Implementar LLM & Agent Clients (Claude / Antigravity / Codex / Spark)

## Critérios de aceite

- [ ] A interface consome exclusivamente os contratos declarados em `api/endpoints.yaml`.
- [ ] Estados de carregamento, vazio e erro estão implementados em todas as telas do fluxo.
- [ ] Contrato `GET /v1/health` implementado e validado contra `api/endpoints.yaml`.
- [ ] Contrato `GET /v1/tools` implementado e validado contra `api/endpoints.yaml`.
- [ ] Contrato `POST /v1/tools/execute` implementado e validado contra `api/endpoints.yaml`.
