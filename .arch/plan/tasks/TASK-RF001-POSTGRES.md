---
id: TASK-RF001-POSTGRES
type: task
title: Implementar autenticação de usuários em PostgreSQL (RF001)
status: pending
priority: must
rank: la
parent: ST-RF001
estimate_h: 8
tier: data
component: PostgreSQL
component_id: node-postgres
dependencies:
  - TASK-POSTGRES-01
requirements:
  - RF001
use_cases:
  - CDU001
source: slice:RF001:node-postgres
order: 8
archived: true
archive_reason: origem removida da arquitetura (slice:RF001:node-postgres)
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:20:14Z"
---

# TASK-RF001-POSTGRES · Implementar autenticação de usuários em PostgreSQL (RF001)

## Critérios de aceite

- [ ] RF001 atendido em PostgreSQL: Autenticação de usuários
- [ ] **Given** um usuário cadastrado com credenciais válidas **When** envia `POST /api/v1/auth/login` **Then** recebe HTTP 200 com `{"token": "<jwt>", "expires_in": 3600}`.
- [ ] **Given** um usuário com senha incorreta **When** envia `POST /api/v1/auth/login` **Then** recebe HTTP 401 sem indicar qual campo falhou.
