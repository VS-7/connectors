---
id: EP-CDU004
type: epic
title: Comunicação via WhatsApp, Slack e Mattermost
status: pending
priority: should
rank: z
requirements:
  - RF005
  - RF006
  - RF007
  - RNF004
use_cases:
  - CDU004
source: usecase:CDU004
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# EP-CDU004 · Comunicação via WhatsApp, Slack e Mattermost

## Descrição

Permite que assistentes inteligentes e o Second Brain interajam diretamente com canais de comunicação corporativos e mensagens instantâneas (WhatsApp, Slack e Mattermost) para envio de alertas, acompanhamento de projetos ou busca de conversas passadas.

## Critérios de aceite

- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
