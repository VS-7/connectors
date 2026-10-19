---
id: TASK-RF006-CORE
type: task
title: Implementar conector Slack com Suporte a User e Bot Tokens em Core Connectors Hub (RF006)
status: completed
priority: should
rank: zzzzzzzV
parent: ST-RF006
estimate_h: 2
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF006
use_cases:
  - CDU004
source: slice:RF006:node-core-hub
order: 28
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF006-CORE · Implementar conector Slack com Suporte a User e Bot Tokens em Core Connectors Hub (RF006)

## Critérios de aceite

- [ ] RF006 atendido em Core Connectors Hub: Conector Slack com Suporte a User e Bot Tokens
- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
