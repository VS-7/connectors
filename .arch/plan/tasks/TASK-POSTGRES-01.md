---
id: TASK-POSTGRES-01
type: task
title: Implementar PostgreSQL (PostgreSQL 16)
status: pending
priority: must
rank: J2
parent: ST-RF001
estimate_h: 12
tier: data
component: PostgreSQL
component_id: node-postgres
requirements:
  - RF001
use_cases:
  - CDU001
source: component:node-postgres
order: 1
archived: true
archive_reason: origem removida da arquitetura (component:node-postgres)
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:20:14Z"
---

# TASK-POSTGRES-01 · Implementar PostgreSQL (PostgreSQL 16)

## Critérios de aceite

- [ ] Migrações idempotentes criam o schema de `PostgreSQL` e podem ser aplicadas duas vezes sem erro.
- [ ] Rollback da última migração restaura o schema anterior sem perda de dados de teste.
