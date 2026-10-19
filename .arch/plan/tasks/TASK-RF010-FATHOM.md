---
id: TASK-RF010-FATHOM
type: task
title: Implementar integração com o Second Brain (OS-Optimizer-Playbook) em Fathom Connector (RF010)
status: completed
priority: must
rank: zzzzzzzzzk
parent: ST-RF010
estimate_h: 2.5
tier: integration
component: Fathom Connector
component_id: node-conn-fathom
dependencies:
  - TASK-FATHOM-02
requirements:
  - RF010
use_cases:
  - CDU003
source: slice:RF010:node-conn-fathom
order: 41
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:53Z"
completed_at: "2026-10-19T01:39:53Z"
---

# TASK-RF010-FATHOM · Implementar integração com o Second Brain (OS-Optimizer-Playbook) em Fathom Connector (RF010)

## Critérios de aceite

- [ ] RF010 atendido em Fathom Connector: Integração com o Second Brain (OS-Optimizer-Playbook)
- [ ] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
