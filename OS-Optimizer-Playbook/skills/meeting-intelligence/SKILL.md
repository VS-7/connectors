---
name: meeting-intelligence
description: "Process raw meeting transcripts (tl:dv, Fireflies, Zoom, Google Meet) into structured executive summaries, action items by owner, settled decisions, and automatic updates to active projects. TRIGGERS: process meeting, process transcript, ata de reuniao, inteligência de reunião, resumir call, extrair tarefas da reunião."
---

# Meeting Intelligence Skill

Takes raw call transcripts or meeting notes and synthesizes them into actionable Second Brain artifacts, linking them to projects, decisions, and customer intelligence.

## Operating Flow

### 1. Ingest & Analyze
- **Origem dos Dados:** O transcript pode ser fornecido diretamente pelo usuário, lido de um arquivo em `Meetings/`, ou obtido automaticamente via conector Fathom (`fathom_get_call_transcript` / `fathom_list_recordings`) do Universal Connectors Hub (`connectors`).
- Identificar participantes, tópicos principais da pauta, decisões tomadas, compromissos e prazos definidos.


### 2. Structure the Output
Format the note according to `[[Meetings/template-meeting]]`:
1. **Executive Summary:** 2–3 dense, clear paragraphs covering what happened, key concerns, and mutual alignment.
2. **Decisions Made:** Bullet points highlighting agreements reached. If a decision is strategic, propose creating a permanent record in `Decisions/`.
3. **Action Items Table:** Clear table with Columns `[Ação, Responsável, Prazo, Projeto Vinculado]`.
4. **ICP & Sales Insights:** Note any customer objections or feature requests to feed `Context/icp.md`.
5. **Raw Transcript Collapse:** Keep the full transcript inside a `<details>` tag at the bottom for searchability without attention rot.

### 3. Update Memory & Active State
- If new tasks relate to existing projects, append the tasks to `Projects/active-projects.md` under the appropriate project heading.
- If a project status changed, update the status column in `Projects/active-projects.md`.
