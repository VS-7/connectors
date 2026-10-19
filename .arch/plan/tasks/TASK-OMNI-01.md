---
id: TASK-OMNI-01
type: task
title: Implementar Omni HTTP Gateway (Go 1.23 / net/http)
status: completed
priority: must
rank: zzzzy
estimate_h: 13
tier: integration
component: Omni HTTP Gateway
component_id: node-omni-http
dependencies:
  - TASK-CORE-01
requirements:
  - RF003
  - RF009
  - RNF003
  - RNF005
use_cases:
  - CDU002
  - CDU006
endpoints:
  - GET /v1/health
  - GET /v1/tools
  - POST /v1/tools/execute
source: component:node-omni-http
order: 13
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:40Z"
completed_at: "2026-10-19T01:39:40Z"
---

# TASK-OMNI-01 · Implementar Omni HTTP Gateway (Go 1.23 / net/http)

## Critérios de aceite

- [ ] Rotas desconhecidas retornam 404 e erros upstream retornam 502 com corpo padronizado.
- [ ] Rate limiting e CORS configurados e cobertos por teste de integração.
- [ ] Contrato `GET /v1/health` implementado e validado contra `api/endpoints.yaml`.
- [ ] Contrato `GET /v1/tools` implementado e validado contra `api/endpoints.yaml`.
- [ ] Contrato `POST /v1/tools/execute` implementado e validado contra `api/endpoints.yaml`.
