# ADR-001 — Arquitetura Hexagonal e Dupla Exposição de Protocolos (MCP e OpenAI Tools)

- **Status:** Aceito
- **Data:** 2026-10-18

## Contexto

Deseja-se criar um hub universal de conectores em Go para integrar ferramentas de comunicação, reuniões e produtividade (WhatsApp, Slack, Mattermost, Google Drive, Gmail e Fathom) com diversos ecossistemas de IA. Existem dois paradigmas predominantes no mercado:
1. Clientes nativos do Model Context Protocol (MCP), como Claude Desktop, Claude Code, Cursor e Antigravity CLI.
2. Clientes baseados no formato clássico de Function Calling / Tool Calling da OpenAI (OpenAI Codex, Spark, LiteLLM e chamadas diretas de API via REST).

Acoplar a lógica dos conectores diretamente à biblioteca de MCP ou a frameworks HTTP causaria duplicação de regras, retrabalho na manutenção e forte dependência de transporte.

## Decisão

Adotamos a **Arquitetura Hexagonal (Ports & Adapters)** com um monorepo em Go:
- **Port Primário (Contrato Universal):** No pacote `internal/core`, definimos a interface `core.Connector` e a struct `core.ToolDefinition` com esquemas baseados em JSON Schema Draft 7 e handlers agnósticos (`func(ctx context.Context, args json.RawMessage) (any, error)`).
- **Adapters de Entrada (Bridges):**
  - `internal/bridge/mcp`: Traduz `core.Connector` para servidor MCP usando `github.com/mark3labs/mcp-go` (binário `cmd/omni-mcp` via stdio e SSE).
  - `internal/bridge/openai`: Traduz `core.Connector` para schemas OpenAI e endpoints REST (`cmd/omni-http` com `GET /v1/tools` e `POST /v1/tools/execute`).
- **Adapters de Saída (Conectores):** Cada conector (`internal/connectors/{whatsapp,slack,mattermost,google,fathom}`) é uma unidade isolada que implementa apenas `core.Connector`.

## Consequências

- **Positivo:** Reutilização total da lógica de negócio e integrações entre Claude, Antigravity, OpenAI e Spark sem alterar uma única linha dos conectores.
- **Positivo:** Possibilidade de compilar tanto um binário único (Super-Server) quanto binários especializados para ambientes com restrição de memória.
- **Negativo:** Pequeno overhead de serialização e desserialização de `json.RawMessage` na ponte entre os protocolos de transporte e os manipuladores internos.
