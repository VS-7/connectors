# Requisitos — connectors

> Documento gerenciado pelo ArchCode Studio. Editável por humanos e por agentes de IA via MCP.

## Visão Geral

O **Universal LLM Connectors Hub** (`connectors`) é uma plataforma centralizada e de alta performance desenvolvida em Go para conectar clientes de Inteligência Artificial — como Claude Desktop, Claude Code, Antigravity CLI, OpenAI Codex, Spark e agentes autônomos — a serviços externos de comunicação, colaboração e produtividade (WhatsApp, Slack, Mattermost, Google Drive, Gmail e Fathom Video), operando de forma nativa e integrada com o **OS-Optimizer-Playbook** (Second Brain & AI OS).

O sistema adota a **Arquitetura Hexagonal (Ports & Adapters)** com camada dupla de exposição de protocolo: os conectores são implementados como adapters de saída agnósticos a modelos de IA, enquanto adapters primários de entrada expõem essas capacidades simultaneamente via **Model Context Protocol (MCP)** e via **OpenAI Function Calling Gateway (REST)**.

## Requisitos Funcionais

### RF001 — Contrato Universal e Registro Hexagonal de Conectores

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-core-hub

O sistema deve fornecer no pacote `internal/core` uma interface agnóstica `core.Connector` (composta por `Namespace()`, `Initialize(ctx)`, `Tools()` e `Close()`) e uma estrutura universal `core.ToolDefinition` (contendo nome, descrição, JSON Schema Draft 7 dos parâmetros e handler de execução `func(ctx context.Context, args json.RawMessage) (any, error)`). O hub deve permitir registrar conectores dinamicamente e despachar chamadas a partir de identificadores padronizados.

### RF002 — Exposição MCP Nativa via stdio e SSE

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-omni-mcp, node-core-hub

O sistema deve implementar uma bridge `internal/bridge/mcp` baseada na biblioteca `github.com/mark3labs/mcp-go`, compilada no binário `cmd/omni-mcp`. A bridge deve converter a lista de `[]core.Connector` em ferramentas MCP registradas nativamente, suportando transporte via `stdio` (JSON-RPC para Claude Code, Antigravity CLI e Claude Desktop) e transporte de rede via `SSE` (Server-Sent Events para agentes distribuídos).

### RF003 — Gateway REST de OpenAI Function Calling

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-omni-http, node-core-hub

O sistema deve disponibilizar um servidor HTTP leve no binário `cmd/omni-http` expondo endpoints REST para agentes que não utilizam MCP nativo (Codex, Spark, LiteLLM e SDKs OpenAI/Anthropic): `GET /v1/tools` (que traduz o catálogo para o formato `tools: [{ "type": "function", "function": {...} }]`) e `POST /v1/tools/execute` (que executa a ferramenta indicada e retorna o resultado em JSON).

### RF004 — Conector Fathom Video para Transcrições e Resumos

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-conn-fathom, node-ext-fathom, node-core-hub

O sistema deve disponibilizar um conector para a API do Fathom Video (`internal/connectors/fathom`), permitindo buscar transcrições completas com timestamps, resumos executivos e pontos de ação gravados em chamadas de reunião através das ferramentas `fathom_get_call_transcript` e `fathom_list_recordings` integrando com a API externa `node-ext-fathom`.

### RF005 — Conector WhatsApp

- **Prioridade:** Média
- **Status:** implemented
- **Componentes:** node-conn-whatsapp, node-sqlite-session, node-core-hub

O sistema deve disponibilizar um conector para WhatsApp (`internal/connectors/whatsapp`) utilizando `go.mau.fi/whatsmeow` com persistência local de sessão em SQLite (`node-sqlite-session`) ou cliente REST para Evolution API. Deve expor ferramentas para envio de mensagens (`whatsapp_send_message`), listagem de conversas recentes (`whatsapp_get_recent_chats`) e busca em histórico de texto.

### RF006 — Conector Slack com Suporte a User e Bot Tokens

- **Prioridade:** Média
- **Status:** implemented
- **Componentes:** node-conn-slack, node-ext-slack, node-core-hub

O sistema deve disponibilizar um conector para Slack (`internal/connectors/slack`) utilizando `github.com/slack-go/slack` integrado com `node-ext-slack`, suportando autenticação por Bot Tokens (`xoxb-`) e User Tokens (`xoxp-`). Deve expor ferramentas para postar mensagens em canais ou threads (`slack_send_message`), consultar histórico recente (`slack_read_channel_history`) e buscar mensagens (`slack_search_messages`).

### RF007 — Conector Mattermost via Client4

- **Prioridade:** Baixa
- **Status:** implemented
- **Componentes:** node-conn-mattermost, node-ext-mattermost, node-core-hub

O sistema deve disponibilizar um conector para Mattermost (`internal/connectors/mattermost`) utilizando `mattermost/mattermost-server/v6/model` conectado a `node-ext-mattermost` com autenticação via Personal Access Token. Deve expor ferramentas para criação de postagens (`mattermost_post_message`), consulta de canais (`mattermost_get_channel_posts`) e pesquisa de mensagens na equipe.

### RF008 — Conector Google Workspace (Gmail e Google Drive)

- **Prioridade:** Média
- **Status:** implemented
- **Componentes:** node-conn-google, node-ext-google, node-core-hub

O sistema deve disponibilizar um conector para Google Workspace (`internal/connectors/google`) conectado a `node-ext-google` utilizando `google.golang.org/api` para Gmail v1 e Google Drive v3, com fluxo OAuth2 local para geração de token com renovação automática. Deve expor ferramentas para buscar e enviar e-mails (`gmail_search_messages`, `gmail_send_message`) e pesquisar arquivos (`drive_search_files`, `drive_get_file_content`).

### RF009 — Ativação Seletiva e Modulação de Ferramentas

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-core-hub, node-omni-mcp, node-omni-http

O sistema deve permitir selecionar dinamicamente quais conectores inicializar no momento do boot via flag CLI (`--enable=fathom,slack,google`) ou variável de ambiente (`ENABLED_CONNECTORS`), evitando expor dezenas de ferramentas desnecessárias e preservando o espaço de contexto da LLM.

### RF010 — Integração com o Second Brain (OS-Optimizer-Playbook)

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-second-brain, node-conn-fathom, node-core-hub

O sistema deve operar em simbiose com o vault do `OS-Optimizer-Playbook` (`node-second-brain`), viabilizando que as transcrições recuperadas pelo conector Fathom sejam transformadas pela skill `meeting-intelligence` em atas estruturadas na pasta `Meetings/` e tarefas sincronizadas no arquivo `Projects/active-projects.md`.

## Requisitos Não Funcionais

### RNF001 — Proteção Estrita do Canal stdio (Logs em os.Stderr)

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-omni-mcp, node-core-hub
- **Categoria:** Confiabilidade
- **Requisitos associados:** RF002

No modo de transporte stdio do MCP, o descritor de arquivo `os.Stdout` deve ser reservado exclusivamente para o protocolo JSON-RPC. Toda e qualquer mensagem de log da aplicação ou de bibliotecas deve ser configurada via `log/slog` para ser emitida exclusivamente em `os.Stderr` ou em arquivo, impedindo corrupção no parser do cliente LLM.

### RNF002 — Token Budgeting e Minificação de DTOs

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-core-hub, node-conn-fathom, node-conn-google, node-conn-slack
- **Categoria:** Desempenho
- **Requisitos associados:** RF004, RF006, RF008

Os adapters de saída nunca devem retornar payloads brutos de APIs externas para a LLM. Todas as respostas devem ser convertidas em DTOs compactos com campos essenciais (id, title, sender, timestamp, snippet) e textos longos devem sofrer truncamento configurável com indicador de omissão, garantindo economia de tokens e evitando degradação de atenção.

### RNF003 — Arquitetura Hexagonal e Desacoplamento Monorepo

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-core-hub, node-omni-mcp, node-omni-http
- **Categoria:** Arquitetura
- **Requisitos associados:** RF001, RF002, RF003

O pacote `internal/core` deve ser completamente independente de protocolos de transporte e de clientes de terceiros. A estrutura do monorepo deve permitir compilar tanto os binários universais (`cmd/omni-mcp`, `cmd/omni-http`) quanto binários individuais especializados por conector sem alteração no código de domínio.

### RNF004 — Gestão Segura de Credenciais e Tokens Locais

- **Prioridade:** Alta
- **Status:** implemented
- **Componentes:** node-conn-google, node-conn-whatsapp, node-sqlite-session, node-core-hub
- **Categoria:** Segurança
- **Requisitos associados:** RF005, RF008

Credenciais sensíveis, tokens de atualização OAuth2 e bancos SQLite de sessão devem ser mantidos fora da árvore do repositório Git, armazenados no diretório do usuário em `~/.config/omni-connectors/` com permissões restritas de sistema de arquivos (`0700` para diretórios e `0600` para arquivos).

### RNF005 — Baixa Latência e Cancelamento via Context

- **Prioridade:** Média
- **Status:** implemented
- **Componentes:** node-core-hub, node-omni-mcp, node-omni-http
- **Categoria:** Desempenho
- **Requisitos associados:** Todos

Todas as invocações de ferramentas devem propagar o `context.Context` até os clientes HTTP/WebSocket externos, com suporte nativo a cancelamento do cliente e limites de timeout configuráveis (5s para consultas simples, 30s para download de arquivos/transcrições extensas).
