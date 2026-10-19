---
id: TASK-RF002-OMNI
type: task
title: Implementar exposição MCP Nativa via stdio e SSE em Omni MCP Server (RF002)
status: completed
priority: must
rank: zzzzzs
parent: ST-RF002
estimate_h: 2.5
tier: integration
component: Omni MCP Server
component_id: node-omni-mcp
dependencies:
  - TASK-OMNI-02
requirements:
  - RF002
use_cases:
  - CDU001
source: slice:RF002:node-omni-mcp
order: 18
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF002-OMNI · Implementar exposição MCP Nativa via stdio e SSE em Omni MCP Server (RF002)

## Critérios de aceite

- [ ] RF002 atendido em Omni MCP Server: Exposição MCP Nativa via stdio e SSE
- [ ] **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- [ ] **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
