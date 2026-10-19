# CDU004 — Comunicação via WhatsApp, Slack e Mattermost

- **Atores:** Agente de IA / Operador
- **Componentes:** node-conn-whatsapp, node-sqlite-session, node-conn-slack, node-ext-slack, node-conn-mattermost, node-ext-mattermost, node-core-hub

- **Complexidade:** high
- **Horas Estimadas:** 24
- **Prioridade:** Média
- **Status:** implemented
- **Requisitos:** RF005, RF006, RF007, RNF004

## Descrição

Permite que assistentes inteligentes e o Second Brain interajam diretamente com canais de comunicação corporativos e mensagens instantâneas (WhatsApp, Slack e Mattermost) para envio de alertas, acompanhamento de projetos ou busca de conversas passadas.

## Pré-condições

- Conectores configurados com tokens respectivos:
  - WhatsApp: Sessão pareada via QR Code ou token de Evolution API.
  - Slack: Token `xoxb-` (Bot) ou `xoxp-` (User).
  - Mattermost: Personal Access Token e URL do servidor.

## Pós-condições

- As mensagens são transmitidas e registradas nas plataformas de destino.
- O histórico pesquisado é devolvido em formato sintetizado para a LLM.

## Fluxo Principal

1. O agente recebe a instrução de notificar um canal ou buscar contexto recente.
2. O conector correspondente é acionado (ex: `slack_send_message` ou `whatsapp_send_message`).
3. O payload é validado contra o esquema da ferramenta.
4. O conector realiza a chamada de rede utilizando a biblioteca especializada em Go (`slack-go`, `whatsmeow` ou `mattermost-server/model`).
5. A resposta de confirmação com ID da mensagem e status é devolvida ao agente.

## Fluxos Alternativos

- **Busca de mensagens anteriores:** O conector recupera as últimas mensagens de um canal ou chat, remove payloads binários ou anexos pesados e retorna apenas texto, autor e horário.

## Exceções

- **Desconexão de Sessão do WhatsApp:** O conector alerta que o dispositivo foi desemparelhado e expõe instrução para reautenticação.
- **Limite de Taxa (Rate Limiting HTTP 429):** O conector aplica backoff exponencial automático antes de retornar erro.

## Regras de Negócio

- Em modo stdio, conexões persistentes (como o WebSocket do whatsmeow) devem ser tratadas de forma desacoplada ou rodar via daemon para não reconectar a cada invocation de tool.
- Credenciais e bancos de sessão SQLite devem ser salvos exclusivamente em `~/.config/omni-connectors/`.

## Critérios de Aceite

- **Given** credenciais válidas do Slack **When** invocada `slack_send_message` com canal e texto **Then** a mensagem é postada no Slack e a tool devolve `{"status": "delivered", "message_id": "<id>"}`.
- **Given** busca em mensagens do Mattermost **When** invocada `mattermost_search_posts` **Then** retorna lista filtrada contendo no máximo 10 snippets compactos.
