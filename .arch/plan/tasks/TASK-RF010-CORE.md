---
id: TASK-RF010-CORE
type: task
title: Implementar integração com o Second Brain (OS-Optimizer-Playbook) em Core Connectors Hub (RF010)
status: completed
priority: must
rank: zzzzzzzzzV
parent: ST-RF010
estimate_h: 2.5
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF010
use_cases:
  - CDU003
source: slice:RF010:node-core-hub
order: 40
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:53Z"
completed_at: "2026-10-19T01:39:53Z"
---

# TASK-RF010-CORE · Implementar integração com o Second Brain (OS-Optimizer-Playbook) em Core Connectors Hub (RF010)

## Critérios de aceite

- [ ] RF010 atendido em Core Connectors Hub: Integração com o Second Brain (OS-Optimizer-Playbook)
- [ ] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
