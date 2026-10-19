---
id: EP-CDU001
type: epic
title: Executar Ferramenta via MCP
status: pending
priority: must
rank: 4l
requirements:
  - RF001
  - RF002
  - RNF001
use_cases:
  - CDU001
source: usecase:CDU001
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:20:14Z"
---

# EP-CDU001 · Executar Ferramenta via MCP

## Descrição

Um cliente compatível com o Model Context Protocol (MCP) conecta-se via `stdio` ou `SSE` ao servidor `omni-mcp`, descobre as ferramentas expostas dinamicamente e executa uma ação enviando parâmetros validados pelo JSON Schema Draft 7.

## Critérios de aceite

- [ ] **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- [ ] **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
