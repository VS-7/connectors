---
id: EP-CDU006
type: epic
title: Inicializar Hub com Ativação Seletiva
status: pending
priority: must
rank: zk
requirements:
  - RF009
  - RNF003
use_cases:
  - CDU006
source: usecase:CDU006
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# EP-CDU006 · Inicializar Hub com Ativação Seletiva

## Descrição

Permite iniciar o servidor MCP ou o gateway HTTP habilitando exclusivamente os conectores desejados para um contexto específico, evitando poluição de contexto e reduzindo o consumo inicial de tokens da LLM.

## Critérios de aceite

- [ ] **Given** o comando `omni-mcp --enable=fathom` **When** o cliente executa `tools/list` **Then** apenas ferramentas com prefixo `fathom_` são listadas.
- [ ] **Given** uma chamada a `GET /v1/health` **When** o gateway foi iniciado com `--enable=slack,mattermost` **Then** o JSON de resposta exibe `enabled_connectors: ["slack", "mattermost"]`.
