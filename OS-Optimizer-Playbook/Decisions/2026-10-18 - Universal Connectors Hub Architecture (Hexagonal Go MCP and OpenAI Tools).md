---
status: active
type: decision
tags: [decision, adr, strategy, architecture, connectors, mcp, second-brain]
date: 2026-10-18
deciders: [Vitor Sergio, Antigravity Agent]
status_decision: Aprovado
---

# Decision: Universal Connectors Hub Architecture (Hexagonal Go, MCP e OpenAI Tools)

## 1. Contexto e Problema

O ecossistema do Second Brain e AI OS (`OS-Optimizer-Playbook`) necessita interagir de forma ativa com plataformas de comunicação e produtividade externas (WhatsApp, Slack, Mattermost, Google Drive, Gmail e Fathom).
Porém, diferentes clientes e motores de IA operam em protocolos distintos:
- Claude Desktop, Claude Code e Antigravity CLI utilizam nativamente o **Model Context Protocol (MCP)** via `stdio` ou `SSE`.
- OpenAI Codex, Spark, LiteLLM e integrações customizadas utilizam **Function Calling / Tool Calling** via HTTP REST.
- Criar servidores isolados ou acoplar os conectores diretamente à biblioteca MCP geraria duplicação de lógica, incompatibilidade de transporte e manutenção insustentável.

## 2. Opções Avaliadas

1. **Servidores MCP Independentes em Node.js / Python:** Criar um script separado para cada ferramenta.
   - *Prós:* Rápido para prototipar individualmente.
   - *Contras:* Alto consumo de memória com múltiplos runtimes em segundo plano, lentidão de boot, falta de suporte direto para clientes sem MCP (Codex/Spark).
2. **Gateway Monolítico REST com tradução HTTP ad-hoc:** Servidor REST tradicional sem suporte a MCP.
   - *Prós:* Simples para agentes via HTTP.
   - *Contras:* Incompatível com Claude Desktop e Antigravity CLI em modo nativo stdio.
3. **Arquitetura Hexagonal (Ports & Adapters) em Go com Camada Dupla de Exposição:**
   - *Prós:* Contrato universal Go (`core.Connector`), binário compilado único de altíssima performance, suporte simultâneo a MCP (`mark3labs/mcp-go`) e REST OpenAI Function Calling, controle total de token budget e proteção estrita do canal stdio.
   - *Contras:* Exige implementação cuidadosa de DTOs enxutos e isolamento de dependências.

## 3. Decisão Tomada & Justificativa

Adotamos a **Opção 3 — Arquitetura Hexagonal em Go com Camada Dupla de Exposição**:
- O núcleo (`internal/core`) define o contrato agnóstico `core.Connector` e `core.ToolDefinition`.
- Dois bridges primários expõem as ferramentas: `cmd/omni-mcp` (stdio/SSE) e `cmd/omni-http` (REST).
- Adapters de saída isolados conectam-se ao Fathom, WhatsApp (whatsmeow / SQLite), Slack (slack-go), Mattermost (Client4) e Google (Gmail/Drive).
- As transcrições do Fathom alimentam diretamente o fluxo de `skills/meeting-intelligence/` e os arquivos em `Meetings/` e `Projects/active-projects.md`.
- Todo o planejamento arquitetural, requisitos (10 RFs, 5 RNFs, 6 CDUs) e tarefas (42 tasks) são gerenciados formalmente via ArchCode Studio CLI/MCP.

## 4. Consequências & Trade-offs

- **Ganhos:**
  - 100% de reutilização das integrações entre qualquer motor de IA (Gemini, Claude, OpenAI, Spark).
  - Boot sub-segundo e consumo de memória inferior a 30 MB.
  - Token budget otimizado com truncamento de payloads, eliminando o "attention rot" e barateando custos de tokens.
  - Logs direcionados a `os.Stderr`, garantindo integridade absoluta do protocolo MCP via stdio.
- **Trade-offs / Riscos:**
  - Whatsmeow requer SQLite local e conexão persistente (deve rodar via daemon ou SSE para não reconectar repetidamente).
  - Necessidade de atualizar tokens OAuth2 locais do Google em `~/.config/omni-connectors/`.
