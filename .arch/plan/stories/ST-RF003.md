---
id: ST-RF003
type: story
title: Gateway REST de OpenAI Function Calling
status: pending
priority: must
rank: zs
parent: EP-CDU002
requirements:
  - RF003
use_cases:
  - CDU002
source: requirement:RF003
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF003 · Gateway REST de OpenAI Function Calling

## Descrição

O sistema deve disponibilizar um servidor HTTP leve no binário `cmd/omni-http` expondo endpoints REST para agentes que não utilizam MCP nativo (Codex, Spark, LiteLLM e SDKs OpenAI/Anthropic): `GET /v1/tools` (que traduz o catálogo para o formato `tools: [{ "type": "function", "function": {...} }]`) e `POST /v1/tools/execute` (que executa a ferramenta indicada e retorna o resultado em JSON).

## Critérios de aceite

- [ ] **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- [ ] **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
