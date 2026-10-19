---
id: TASK-RF007-CORE
type: task
title: Implementar conector Mattermost via Client4 em Core Connectors Hub (RF007)
status: completed
priority: could
rank: zzzzzzzw
parent: ST-RF007
estimate_h: 2
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF007
use_cases:
  - CDU004
source: slice:RF007:node-core-hub
order: 31
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF007-CORE · Implementar conector Mattermost via Client4 em Core Connectors Hub (RF007)

## Critérios de aceite

- [ ] RF007 atendido em Core Connectors Hub: Conector Mattermost via Client4
- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
