---
id: TASK-RF002-CORE
type: task
title: Implementar exposição MCP Nativa via stdio e SSE em Core Connectors Hub (RF002)
status: completed
priority: must
rank: zzzzzzzzzz
parent: ST-RF002
estimate_h: 2.5
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF002
use_cases:
  - CDU001
source: slice:RF002:node-core-hub
order: 19
created_at: "2026-10-19T01:20:33Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF002-CORE · Implementar exposição MCP Nativa via stdio e SSE em Core Connectors Hub (RF002)

## Critérios de aceite

- [ ] RF002 atendido em Core Connectors Hub: Exposição MCP Nativa via stdio e SSE
- [ ] **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- [ ] **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
