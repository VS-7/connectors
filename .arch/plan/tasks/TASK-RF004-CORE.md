---
id: TASK-RF004-CORE
type: task
title: Implementar conector Fathom Video para Transcrições e Resumos em Core Connectors Hub (RF004)
status: completed
priority: must
rank: zzzzzzV
parent: ST-RF004
estimate_h: 2.5
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF004
use_cases:
  - CDU003
source: slice:RF004:node-core-hub
order: 22
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF004-CORE · Implementar conector Fathom Video para Transcrições e Resumos em Core Connectors Hub (RF004)

## Critérios de aceite

- [ ] RF004 atendido em Core Connectors Hub: Conector Fathom Video para Transcrições e Resumos
- [ ] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
