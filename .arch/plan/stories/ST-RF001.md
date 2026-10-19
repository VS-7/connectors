---
id: ST-RF001
type: story
title: Contrato Universal e Registro Hexagonal de Conectores
status: pending
priority: must
rank: 9W
parent: EP-CDU001
requirements:
  - RF001
use_cases:
  - CDU001
  - CDU002
source: requirement:RF001
created_at: "2026-10-19T01:03:44Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF001 · Contrato Universal e Registro Hexagonal de Conectores

## Descrição

O sistema deve fornecer no pacote `internal/core` uma interface agnóstica `core.Connector` (composta por `Namespace()`, `Initialize(ctx)`, `Tools()` e `Close()`) e uma estrutura universal `core.ToolDefinition` (contendo nome, descrição, JSON Schema Draft 7 dos parâmetros e handler de execução `func(ctx context.Context, args json.RawMessage) (any, error)`). O hub deve permitir registrar conectores dinamicamente e despachar chamadas a partir de identificadores padronizados.

## Critérios de aceite

- [ ] **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- [ ] **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
- [ ] **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- [ ] **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
