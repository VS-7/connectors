---
status: active
type: folder-index
tags: [meetings, intelligence, layer-2]
date: 2026-10-18
---

# Layer 2 — Meeting Intelligence Hub

**Purpose:** Repositório de notas, atas e inteligência extraída de conversas com clientes, parceiros e equipe.

## Fluxo Operacional
1. Conclua uma reunião gravada (tl:dv, Fireflies, Zoom, Google Meet).
2. Salve ou cole a transcrição em um arquivo nesta pasta: `Meetings/YYYY-MM-DD - [Nome da Reunião].md`.
3. Execute a Skill `skills/meeting-intelligence/` para processar a transcrição.
4. A Skill extrai:
   - Resumo executivo (2-3 parágrafos)
   - Decisões tomadas (com links para `[[Decisions/]]` quando estruturais)
   - Próximos passos e tarefas atribuídas por pessoa
   - Atualizações automáticas em `[[active-projects]]`

## Arquivos Modelo
- [[template-meeting]] — Estrutura padrão para documentação de reuniões.
