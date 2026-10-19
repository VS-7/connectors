# CDU001 — Executar Ferramenta via MCP

- **Atores:** Cliente MCP (Claude Desktop, Claude Code, Antigravity CLI)
- **Componentes:** node-llm-clients, node-omni-mcp, node-core-hub
- **Complexidade:** medium
- **Horas Estimadas:** 16
- **Prioridade:** Alta
- **Status:** implemented
- **Requisitos:** RF001, RF002, RNF001

## Descrição

Um cliente compatível com o Model Context Protocol (MCP) conecta-se via `stdio` ou `SSE` ao servidor `omni-mcp`, descobre as ferramentas expostas dinamicamente e executa uma ação enviando parâmetros validados pelo JSON Schema Draft 7.

## Pré-condições

- O binário `omni-mcp` está iniciado ou configurado no cliente MCP (ex: `claude mcp add` ou `~/.cursor/mcp.json`).
- O conector requisitado está devidamente inicializado e com credenciais válidas.

## Pós-condições

- A ferramenta é executada com sucesso e o resultado textual JSON é retornado ao cliente no padrão `mcp.CallToolResult`.
- Todo o tráfego de log é canalizado para `os.Stderr`, mantendo o `os.Stdout` íntegro.

## Fluxo Principal

1. O cliente MCP envia a mensagem JSON-RPC `tools/list` pelo canal `stdio` ou conexão SSE.
2. O servidor `omni-mcp` itera pelos conectores habilitados e responde com o catálogo de ferramentas e seus esquemas de entrada.
3. O LLM decide chamar uma ferramenta (ex: `fathom_get_call_transcript`) e envia `tools/call` com os argumentos JSON correspondentes.
4. O bridge `internal/bridge/mcp` intercepta a chamada, valida os parâmetros e delega para o `Handler` agnóstico em `core.ToolDefinition`.
5. O conector executa a operação e retorna o resultado estruturado.
6. O bridge serializa o resultado em JSON indentado e devolve um `mcp.CallToolResult` para o cliente.

## Fluxos Alternativos

- **Transporte via SSE:** O cliente se conecta por HTTP SSE (`/mcp/sse`); as mensagens fluem por HTTP POST mantendo a mesma semântica de despacho.

## Exceções

- **Parâmetros inválidos:** O conector falha no unmarshal dos argumentos; o bridge retorna `mcp.NewToolResultError` com a mensagem descritiva sem derrubar o processo.
- **Falha na API externa (timeout ou 401/403):** O erro é capturado e retornado de forma clara para o LLM tentar auto-correção.

## Regras de Negócio

- A saída padrão `os.Stdout` nunca deve receber bytes fora da especificação JSON-RPC.
- Todas as mensagens de erro retornadas para o LLM devem ser amigáveis e concisas.

## Critérios de Aceite

- **Given** um cliente MCP conectado via stdio **When** envia `tools/call` com `{name: "fathom_get_call_transcript", arguments: {"recording_id": "rec_123"}}` **Then** recebe `mcp.CallToolResult` contendo o resumo e transcrição no formato textual.
- **Given** uma biblioteca externa gerando logs durante a execução **When** emite texto de debug **Then** todo o conteúdo é direcionado para `os.Stderr` e o cliente MCP não desconecta.
