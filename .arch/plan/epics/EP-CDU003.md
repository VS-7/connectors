---
id: EP-CDU003
type: epic
title: Ingerir Reunião do Fathom no Second Brain
status: pending
priority: must
rank: "y"
requirements:
  - RF004
  - RF010
  - RNF002
use_cases:
  - CDU003
source: usecase:CDU003
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# EP-CDU003 · Ingerir Reunião do Fathom no Second Brain

## Descrição

O conector Fathom busca gravações recentes, transcrições com falantes e resumos gerados pela IA do Fathom, disponibilizando os dados em DTOs enxutos para que a skill `meeting-intelligence` do `OS-Optimizer-Playbook` processe a ata, extraia decisões e sincronize tarefas em `Projects/active-projects.md`.

## Critérios de aceite

- [ ] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
