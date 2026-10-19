---
id: ST-RF007
type: story
title: Conector Mattermost via Client4
status: pending
priority: could
rank: zzV
parent: EP-CDU004
requirements:
  - RF007
use_cases:
  - CDU004
source: requirement:RF007
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF007 · Conector Mattermost via Client4

## Descrição

O sistema deve disponibilizar um conector para Mattermost (`internal/connectors/mattermost`) utilizando `mattermost/mattermost-server/v6/model` conectado a `node-ext-mattermost` com autenticação via Personal Access Token. Deve expor ferramentas para criação de postagens (`mattermost_post_message`), consulta de canais (`mattermost_get_channel_posts`) e pesquisa de mensagens na equipe.

## Critérios de aceite

- [ ] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
