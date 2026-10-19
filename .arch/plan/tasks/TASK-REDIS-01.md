---
id: TASK-REDIS-01
type: task
title: Implementar Redis (Redis 7)
status: pending
rank: Nn
estimate_h: 5
tier: data
component: Redis
component_id: node-redis
requirements:
  - RNF002
source: component:node-redis
order: 2
archived: true
archive_reason: origem removida da arquitetura (component:node-redis)
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:20:14Z"
---

# TASK-REDIS-01 · Implementar Redis (Redis 7)

## Critérios de aceite

- [ ] Chaves de `Redis` têm TTL explícito e a aplicação funciona (degradada) com o cache indisponível.
