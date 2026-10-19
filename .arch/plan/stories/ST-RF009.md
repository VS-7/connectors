---
id: ST-RF009
type: story
title: Ativação Seletiva e Modulação de Ferramentas
status: pending
priority: must
rank: zzs
parent: EP-CDU006
requirements:
  - RF009
use_cases:
  - CDU006
source: requirement:RF009
created_at: "2026-10-19T01:20:14Z"
updated_at: "2026-10-19T01:20:14Z"
---

# ST-RF009 · Ativação Seletiva e Modulação de Ferramentas

## Descrição

O sistema deve permitir selecionar dinamicamente quais conectores inicializar no momento do boot via flag CLI (`--enable=fathom,slack,google`) ou variável de ambiente (`ENABLED_CONNECTORS`), evitando expor dezenas de ferramentas desnecessárias e preservando o espaço de contexto da LLM.

## Critérios de aceite

- [ ] **Given** o comando `omni-mcp --enable=fathom` **When** o cliente executa `tools/list` **Then** apenas ferramentas com prefixo `fathom_` são listadas.
- [ ] **Given** uma chamada a `GET /v1/health` **When** o gateway foi iniciado com `--enable=slack,mattermost` **Then** o JSON de resposta exibe `enabled_connectors: ["slack", "mattermost"]`.
