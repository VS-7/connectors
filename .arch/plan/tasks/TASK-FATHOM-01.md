---
id: TASK-FATHOM-01
type: task
title: Implementar Fathom Video API (HTTPS REST API)
status: completed
priority: must
sprint: 1
rank: zzzV
parent: ST-RF004
estimate_h: 9.5
tier: integration
component: Fathom Video API
component_id: node-ext-fathom
requirements:
  - RF004
use_cases:
  - CDU003
source: component:node-ext-fathom
order: 3
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:40Z"
completed_at: "2026-10-19T01:39:40Z"
---

# TASK-FATHOM-01 · Implementar Fathom Video API (HTTPS REST API)

## Critérios de aceite

- [ ] Integração com `Fathom Video API` possui timeout, retry com backoff e client fake para testes.
- [ ] Credenciais vêm de variáveis de ambiente — nunca versionadas no repositório.
