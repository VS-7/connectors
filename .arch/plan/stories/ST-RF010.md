---
id: ST-RF010
type: story
title: Integração com o Second Brain (OS-Optimizer-Playbook)
status: pending
priority: must
rank: zzw
parent: EP-CDU003
requirements:
  - RF010
use_cases:
  - CDU003
source: requirement:RF010
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF010 · Integração com o Second Brain (OS-Optimizer-Playbook)

## Descrição

O sistema deve operar em simbiose com o vault do `OS-Optimizer-Playbook` (`node-second-brain`), viabilizando que as transcrições recuperadas pelo conector Fathom sejam transformadas pela skill `meeting-intelligence` em atas estruturadas na pasta `Meetings/` e tarefas sincronizadas no arquivo `Projects/active-projects.md`.

## Critérios de aceite

- [ ] **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- [ ] **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
