---
id: TASK-RF005-WHATSAPP
type: task
title: Implementar conector WhatsApp em WhatsApp Connector (RF005)
status: completed
priority: should
rank: zzzzzzy
parent: ST-RF005
estimate_h: 2
tier: integration
component: WhatsApp Connector
component_id: node-conn-whatsapp
dependencies:
  - TASK-WHATSAPP-01
requirements:
  - RF005
use_cases:
  - CDU004
source: slice:RF005:node-conn-whatsapp
order: 26
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF005-WHATSAPP · Implementar conector WhatsApp em WhatsApp Connector (RF005)

## Critérios de aceite

- [ ] RF005 atendido em WhatsApp Connector: Conector WhatsApp
- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
