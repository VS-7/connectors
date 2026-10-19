---
id: TASK-RF004-FATHOM-2
type: task
title: Implementar conector Fathom Video para Transcrições e Resumos em Fathom Video API (RF004)
status: completed
priority: must
rank: zzzzzzs
parent: ST-RF004
estimate_h: 2.5
tier: integration
component: Fathom Video API
component_id: node-ext-fathom
dependencies:
  - TASK-FATHOM-01
requirements:
  - RF004
use_cases:
  - CDU003
source: slice:RF004:node-ext-fathom
order: 24
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF004-FATHOM-2 · Implementar conector Fathom Video para Transcrições e Resumos em Fathom Video API (RF004)

## Critérios de aceite

- [ ] RF004 atendido em Fathom Video API: Conector Fathom Video para Transcrições e Resumos
- [ ] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
