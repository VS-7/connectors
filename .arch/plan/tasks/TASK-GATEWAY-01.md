---
id: TASK-GATEWAY-01
type: task
title: Implementar API Gateway (Traefik)
status: pending
rank: XJ
estimate_h: 13
tier: integration
component: API Gateway
component_id: node-api-gateway
dependencies:
  - TASK-CORE-01
requirements:
  - RNF001
endpoints:
  - POST /api/v1/auth/login
  - GET /api/v1/me
source: component:node-api-gateway
order: 4
archived: true
archive_reason: origem removida da arquitetura (component:node-api-gateway)
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:20:14Z"
---

# TASK-GATEWAY-01 · Implementar API Gateway (Traefik)

## Critérios de aceite

- [ ] Rotas desconhecidas retornam 404 e erros upstream retornam 502 com corpo padronizado.
- [ ] Rate limiting e CORS configurados e cobertos por teste de integração.
- [ ] Contrato `POST /api/v1/auth/login` implementado e validado contra `api/endpoints.yaml`.
- [ ] Contrato `GET /api/v1/me` implementado e validado contra `api/endpoints.yaml`.
