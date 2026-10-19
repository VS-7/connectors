---
id: TASK-RF008-GOOGLE
type: task
title: Implementar conector Google Workspace (Gmail e Google Drive) em Google Workspace Connector (RF008)
status: completed
priority: should
rank: zzzzzzzzk
parent: ST-RF008
estimate_h: 2
tier: integration
component: Google Workspace Connector
component_id: node-conn-google
dependencies:
  - TASK-GOOGLE-02
requirements:
  - RF008
use_cases:
  - CDU005
source: slice:RF008:node-conn-google
order: 35
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF008-GOOGLE · Implementar conector Google Workspace (Gmail e Google Drive) em Google Workspace Connector (RF008)

## Critérios de aceite

- [ ] RF008 atendido em Google Workspace Connector: Conector Google Workspace (Gmail e Google Drive)
- [ ] **Given** token de autenticação Google válido **When** invocada `gmail_search_messages` com uma consulta **Then** retorna lista de e-mails em texto puro sem tags HTML.
- [ ] **Given** uma requisição de busca no Drive **When** executada com sucesso **Then** retorna IDs e títulos dos arquivos encontrados em formato JSON conciso.
