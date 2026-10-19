---
id: TASK-E2E-01
type: task
title: Testes ponta a ponta dos casos de uso
status: completed
rank: v6
estimate_h: 24
tier: devops
dependencies:
  - TASK-CORE-01
  - TASK-FATHOM-01
  - TASK-FATHOM-02
  - TASK-GOOGLE-01
  - TASK-GOOGLE-02
  - TASK-LLM-01
  - TASK-MATTERMO-01
  - TASK-MATTERMO-02
  - TASK-OMNI-01
  - TASK-OMNI-02
  - TASK-OS-01
  - TASK-RF001-CORE
  - TASK-RF002-CORE
  - TASK-RF002-OMNI
  - TASK-RF003-CORE
  - TASK-RF003-OMNI
  - TASK-RF004-CORE
  - TASK-RF004-FATHOM
  - TASK-RF004-FATHOM-2
  - TASK-RF005-CORE
  - TASK-RF005-SQLITE
  - TASK-RF005-WHATSAPP
  - TASK-RF006-CORE
  - TASK-RF006-SLACK
  - TASK-RF006-SLACK-2
  - TASK-RF007-CORE
  - TASK-RF007-MATTERMO
  - TASK-RF007-MATTERMO-2
  - TASK-RF008-CORE
  - TASK-RF008-GOOGLE
  - TASK-RF008-GOOGLE-2
  - TASK-RF009-CORE
  - TASK-RF009-OMNI
  - TASK-RF009-OMNI-2
  - TASK-RF010-CORE
  - TASK-RF010-FATHOM
  - TASK-RF010-OS
  - TASK-SLACK-01
  - TASK-SLACK-02
  - TASK-SQLITE-01
  - TASK-WHATSAPP-01
source: e2e
order: 43
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:39:53Z"
completed_at: "2026-10-19T01:39:53Z"
---

# TASK-E2E-01 · Testes ponta a ponta dos casos de uso

## Critérios de aceite

- [ ] [CDU001] **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- [ ] [CDU001] **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
- [ ] [CDU002] **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- [ ] [CDU002] **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
- [ ] [CDU003] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] [CDU003] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
- [ ] [CDU004] **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- [ ] [CDU004] **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
- [ ] [CDU005] **Given** token de autenticação Google válido **When** invocada `gmail_search_messages` com uma consulta **Then** retorna lista de e-mails em texto puro sem tags HTML.
- [ ] [CDU005] **Given** uma requisição de busca no Drive **When** executada com sucesso **Then** retorna IDs e títulos dos arquivos encontrados em formato JSON conciso.
- [ ] [CDU006] **Given** o comando `omni-mcp --enable=fathom` **When** o cliente executa `tools/list` **Then** apenas ferramentas com prefixo `fathom_` são listadas.
- [ ] [CDU006] **Given** uma chamada a `GET /v1/health` **When** o gateway foi iniciado com `--enable=slack,mattermost` **Then** o JSON de resposta exibe `enabled_connectors: ["slack", "mattermost"]`.
