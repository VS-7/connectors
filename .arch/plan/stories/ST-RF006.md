---
id: ST-RF006
type: story
title: Conector Slack com Suporte a User e Bot Tokens
status: pending
priority: should
rank: zz
parent: EP-CDU004
requirements:
  - RF006
use_cases:
  - CDU004
source: requirement:RF006
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF006 · Conector Slack com Suporte a User e Bot Tokens

## Descrição

O sistema deve disponibilizar um conector para Slack (`internal/connectors/slack`) utilizando `github.com/slack-go/slack` integrado com `node-ext-slack`, suportando autenticação por Bot Tokens (`xoxb-`) e User Tokens (`xoxp-`). Deve expor ferramentas para postar mensagens em canais ou threads (`slack_send_message`), consultar histórico recente (`slack_read_channel_history`) e buscar mensagens (`slack_search_messages`).

## Critérios de aceite

- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
