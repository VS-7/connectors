---
status: active
type: projects-dashboard
tags: [projects, active, roadmap]
date: 2026-10-18
---

# Active Projects Dashboard

## 🎯 Prioridades da Semana / Sprint

| Projeto | Responsável | Status | Próximo Marco | Bloqueios |
|---|---|---|---|---|
| **Setup do AI OS & Second Brain** | Operador | Em Progresso | Finalizar preenchimento da Layer 1 | Nenhum |
| **Pipeline de Inteligência de Reuniões** | Operador | Em Progresso | Integrar conector Fathom com `skills/meeting-intelligence/` | Nenhum |
| **Universal Connectors Hub (Go)** | Operador / IA | Concluído | Binários `omni-mcp` e `omni-http` operacionais | Nenhum |

---

## 📋 Detalhamento dos Projetos Ativos

### 1. Setup do AI OS & Second Brain
- **Objetivo:** Estabelecer a fundação operacional no Gemini / Antigravity / Claude para operar o negócio com agentes autônomos.
- **Contexto Vinculado:** `[[company]]`, `[[identity]]`, `[[tech-stack]]`.
- **Tarefas:**
  - [x] Ajustar OS Optimizer para compatibilidade 100% com Gemini, Spark e Antigravity
  - [ ] Personalizar `Context/identity.md` com voz e regras pessoais
  - [ ] Personalizar `Context/company.md` e `Context/icp.md`
  - [ ] Rodar primeira auditoria com OS Optimizer (`skills/optimizer`)

### 2. Pipeline de Inteligência de Reuniões
- **Objetivo:** Toda reunião gravada pelo Fathom / tl:dv / Fireflies gera ata, tarefas e atualiza este painel em segundos via conector universal.
- **Tarefas:**
  - [x] Configurar Skill `skills/meeting-intelligence/`
  - [x] Integrar chamada de ferramenta `fathom_get_call_transcript` via `connectors`
  - [ ] Testar fluxo com uma transcrição real

### 3. Universal Connectors Hub (Go / MCP / OpenAI Tools)
- **Objetivo:** Super-Server em Go desacoplado via Arquitetura Hexagonal para conectar Claude, Antigravity, OpenAI e Spark a WhatsApp, Slack, Mattermost, Google e Fathom.
- **Contexto Vinculado:** `[[tech-stack]]`, `[[Decisions/2026-10-18 - Universal Connectors Hub Architecture (Hexagonal Go MCP and OpenAI Tools)]]`.
- **Planejamento ArchCode:** Backlog sincronizado em `docs/ai-prd.md` e `.arch/tasks.json` (100% concluído).
- **Tarefas:**
  - [x] Especificação de RF, RNF, CDUs e ADRs com ArchCode Studio CLI
  - [x] Implementar contrato `core.Connector` e `core.ToolDefinition` em `internal/core`
  - [x] Implementar bridge MCP (`internal/bridge/mcp`) e binário `cmd/omni-mcp`
  - [x] Implementar gateway REST OpenAI (`internal/bridge/openai`) e binário `cmd/omni-http`
  - [x] Implementar conector Fathom (`internal/connectors/fathom`) com token budgeting
  - [x] Implementar conectores WhatsApp, Slack, Mattermost e Google Workspace


