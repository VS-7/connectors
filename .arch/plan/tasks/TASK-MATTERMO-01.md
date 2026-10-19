---
id: TASK-MATTERMO-01
type: task
title: Implementar Mattermost Server API (Client4 REST API)
status: completed
priority: could
sprint: 1
rank: zzzy
parent: ST-RF007
estimate_h: 9.5
tier: integration
component: Mattermost Server API
component_id: node-ext-mattermost
requirements:
  - RF007
use_cases:
  - CDU004
source: component:node-ext-mattermost
order: 7
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:40Z"
completed_at: "2026-10-19T01:39:40Z"
---

# TASK-MATTERMO-01 · Implementar Mattermost Server API (Client4 REST API)

## Critérios de aceite

- [ ] Integração com `Mattermost Server API` possui timeout, retry com backoff e client fake para testes.
- [ ] Credenciais vêm de variáveis de ambiente — nunca versionadas no repositório.
