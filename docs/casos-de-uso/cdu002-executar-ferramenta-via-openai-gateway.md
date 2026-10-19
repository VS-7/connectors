# CDU002 — Executar Ferramenta via OpenAI Gateway

- **Atores:** Cliente Function Calling (OpenAI Codex, Spark, LiteLLM, SDKs)
- **Componentes:** node-llm-clients, node-omni-http, node-core-hub
- **Complexidade:** medium
- **Horas Estimadas:** 12
- **Prioridade:** Alta
- **Status:** implemented
- **Requisitos:** RF001, RF003

## Descrição

Aplicações e agentes que não possuem suporte nativo ao protocolo MCP consomem o catálogo universal de ferramentas através de endpoints REST padronizados no formato OpenAI Tool Calling (`GET /v1/tools` e `POST /v1/tools/execute`).

## Pré-condições

- O gateway `omni-http` está em execução e escutando na porta configurada (padrão `:8080`).
- Os conectores solicitados estão ativos.

## Pós-condições

- As ferramentas são expostas como functions no formato OpenAI JSON Schema.
- A requisição `POST /v1/tools/execute` aciona o conector e devolve a resposta em formato JSON.

## Fluxo Principal

1. O cliente HTTP envia `GET /v1/tools`.
2. O gateway `omni-http` lê os `core.ToolDefinition` de todos os conectores ativos e monta a lista `tools: [{"type": "function", "function": {"name": t.Name, "description": t.Description, "parameters": t.InputSchema}}]`.
3. O modelo de linguagem gera uma chamada de função correspondente.
4. O cliente envia `POST /v1/tools/execute` com o payload JSON `{ "name": "slack_send_message", "arguments": { "channel": "#general", "message": "Deploy concluído" } }`.
5. O gateway localiza o handler no mapa em memória e despacha a execução.
6. O resultado retornado pelo conector é empacotado em `{ "result": <payload>, "error": null }` com status HTTP 200.

## Fluxos Alternativos

- **Verificação de Healthcheck:** O cliente pode consultar `GET /v1/health` para checar conectores operacionais antes de disparar execuções.

## Exceções

- **Ferramenta Inexistente:** Se o nome informado não corresponder a nenhuma ferramenta registrada, o gateway retorna HTTP 404 com `{ "error": "ferramenta não encontrada" }`.
- **Erro interno de execução:** Se o conector retornar um erro, o gateway responde HTTP 500 com `{ "error": err.Error() }`.

## Regras de Negócio

- O gateway deve suportar CORS se configurado para permitir acesso via browser ou ferramentas web.
- O formato de parâmetros em `parameters` deve ser estritamente compatível com JSON Schema Draft 7.

## Critérios de Aceite

- **Given** o servidor HTTP em execução **When** recebe `GET /v1/tools` **Then** retorna status 200 com array `tools` onde cada item possui `type: "function"` e o esquema dos parâmetros.
- **Given** uma requisição `POST /v1/tools/execute` com argumentos válidos **When** processada pelo gateway **Then** retorna HTTP 200 e o payload JSON produzido pelo conector.
