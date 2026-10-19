---
id: TASK-RF006-SLACK
type: task
title: Implementar conector Slack com Suporte a User e Bot Tokens em Slack Connector (RF006)
status: completed
priority: should
rank: zzzzzzzk
parent: ST-RF006
estimate_h: 2
tier: integration
component: Slack Connector
component_id: node-conn-slack
dependencies:
  - TASK-SLACK-02
requirements:
  - RF006
use_cases:
  - CDU004
source: slice:RF006:node-conn-slack
order: 29
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF006-SLACK · Implementar conector Slack com Suporte a User e Bot Tokens em Slack Connector (RF006)

## Critérios de aceite

- [ ] RF006 atendido em Slack Connector: Conector Slack com Suporte a User e Bot Tokens
- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
