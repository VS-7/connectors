---
id: EP-CDU002
type: epic
title: Executar Ferramenta via OpenAI Gateway
status: pending
priority: must
rank: x
requirements:
  - RF001
  - RF003
use_cases:
  - CDU002
source: usecase:CDU002
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# EP-CDU002 · Executar Ferramenta via OpenAI Gateway

## Descrição

Aplicações e agentes que não possuem suporte nativo ao protocolo MCP consomem o catálogo universal de ferramentas através de endpoints REST padronizados no formato OpenAI Tool Calling (`GET /v1/tools` e `POST /v1/tools/execute`).

## Critérios de aceite

- [ ] **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- [ ] **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
