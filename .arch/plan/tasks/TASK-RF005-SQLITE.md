---
id: TASK-RF005-SQLITE
type: task
title: Implementar conector WhatsApp em SQLite Session Store (RF005)
status: completed
priority: should
rank: zzzzzzz
parent: ST-RF005
estimate_h: 2
tier: data
component: SQLite Session Store
component_id: node-sqlite-session
dependencies:
  - TASK-SQLITE-01
requirements:
  - RF005
use_cases:
  - CDU004
source: slice:RF005:node-sqlite-session
order: 27
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF005-SQLITE · Implementar conector WhatsApp em SQLite Session Store (RF005)

## Critérios de aceite

- [ ] RF005 atendido em SQLite Session Store: Conector WhatsApp
- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
