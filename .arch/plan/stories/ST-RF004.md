---
id: ST-RF004
type: story
title: Conector Fathom Video para Transcrições e Resumos
status: pending
priority: must
rank: zw
parent: EP-CDU003
requirements:
  - RF004
use_cases:
  - CDU003
source: requirement:RF004
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF004 · Conector Fathom Video para Transcrições e Resumos

## Descrição

O sistema deve disponibilizar um conector para a API do Fathom Video (`internal/connectors/fathom`), permitindo buscar transcrições completas com timestamps, resumos executivos e pontos de ação gravados em chamadas de reunião através das ferramentas `fathom_get_call_transcript` e `fathom_list_recordings` integrando com a API externa `node-ext-fathom`.

## Critérios de aceite

- [ ] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
