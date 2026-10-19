---
id: TASK-RF001-CORE
type: task
title: Implementar contrato Universal e Registro Hexagonal de Conectores em Core Connectors Hub (RF001)
status: completed
priority: must
rank: zzzzzzzzzy
parent: ST-RF001
estimate_h: 11.5
tier: backend
component: Core Connectors Hub
component_id: node-core-hub
dependencies:
  - TASK-CORE-01
requirements:
  - RF001
use_cases:
  - CDU001
  - CDU002
source: slice:RF001:node-core-hub
order: 17
created_at: "2026-10-19T01:20:33Z"
updated_at: "2026-10-19T01:39:52Z"
completed_at: "2026-10-19T01:39:52Z"
---

# TASK-RF001-CORE · Implementar contrato Universal e Registro Hexagonal de Conectores em Core Connectors Hub (RF001)

## Critérios de aceite

- [ ] RF001 atendido em Core Connectors Hub: Contrato Universal e Registro Hexagonal de Conectores
- [ ] **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- [ ] **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
- [ ] **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- [ ] **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
