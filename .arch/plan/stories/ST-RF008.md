---
id: ST-RF008
type: story
title: Conector Google Workspace (Gmail e Google Drive)
status: pending
priority: should
rank: zzk
parent: EP-CDU005
requirements:
  - RF008
use_cases:
  - CDU005
source: requirement:RF008
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF008 · Conector Google Workspace (Gmail e Google Drive)

## Descrição

O sistema deve disponibilizar um conector para Google Workspace (`internal/connectors/google`) conectado a `node-ext-google` utilizando `google.golang.org/api` para Gmail v1 e Google Drive v3, com fluxo OAuth2 local para geração de token com renovação automática. Deve expor ferramentas para buscar e enviar e-mails (`gmail_search_messages`, `gmail_send_message`) e pesquisar arquivos (`drive_search_files`, `drive_get_file_content`).

## Critérios de aceite

- [ ] **Given** token de autenticação Google válido **When** invocada `gmail_search_messages` com uma consulta **Then** retorna lista de e-mails em texto puro sem tags HTML.
- [ ] **Given** uma requisição de busca no Drive **When** executada com sucesso **Then** retorna IDs e títulos dos arquivos encontrados em formato JSON conciso.
