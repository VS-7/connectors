---
id: ST-RF002
type: story
title: Exposição MCP Nativa via stdio e SSE
status: pending
priority: must
rank: EH
parent: EP-CDU001
requirements:
  - RF002
use_cases:
  - CDU001
source: requirement:RF002
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF002 · Exposição MCP Nativa via stdio e SSE

## Descrição

O sistema deve implementar uma bridge `internal/bridge/mcp` baseada na biblioteca `github.com/mark3labs/mcp-go`, compilada no binário `cmd/omni-mcp`. A bridge deve converter a lista de `[]core.Connector` em ferramentas MCP registradas nativamente, suportando transporte via `stdio` (JSON-RPC para Claude Code, Antigravity CLI e Claude Desktop) e transporte de rede via `SSE` (Server-Sent Events para agentes distribuídos).

## Critérios de aceite

- [ ] **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- [ ] **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
