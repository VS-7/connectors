---
id: ST-RF005
type: story
title: Conector WhatsApp
status: pending
priority: should
rank: zy
parent: EP-CDU004
requirements:
  - RF005
use_cases:
  - CDU004
source: requirement:RF005
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF005 · Conector WhatsApp

## Descrição

O sistema deve disponibilizar um conector para WhatsApp (`internal/connectors/whatsapp`) utilizando `go.mau.fi/whatsmeow` com persistência local de sessão em SQLite (`node-sqlite-session`) ou cliente REST para Evolution API. Deve expor ferramentas para envio de mensagens (`whatsapp_send_message`), listagem de conversas recentes (`whatsapp_get_recent_chats`) e busca em histórico de texto.

## Critérios de aceite

- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
