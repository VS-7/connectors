# Conector WhatsApp: guia de uso

O conector usa o [whatsmeow](https://github.com/tulir/whatsmeow) como **aparelho conectado** da sua conta
(igual ao WhatsApp Desktop). A sessão fica em `~/.config/omni-connectors/whatsapp_session.db`. O histórico
fica em `~/.config/omni-connectors/whatsapp_messages.db`, separado da sessão. Nada vai para servidores de
terceiros: a transcrição de áudio também é local.

> **Risco de banimento.** Clientes não oficiais violam os termos do WhatsApp. O modo seguro abaixo faz o
> tráfego parecer o de um cliente comum e reduz muito o risco, mas não o elimina. Para envios em escala,
> use a API oficial (WhatsApp Business).

## 1. Parear
```sh
go build -o bin/omni-wa ./cmd/omni-wa
./bin/omni-wa sync            # abre data/whatsapp/qr.html; escaneie em Aparelhos conectados
```
- Mantenha o WhatsApp **aberto no celular** até os lotes de histórico pararem.
- `-full` pede o histórico completo. É pesado para o celular: use só no primeiro pareamento.
- `-phone 55DDDNUMERO` pareia por código em vez de QR.

Se as primeiras mensagens do seu celular aparecerem como `mensagem não decriptada`, mande uma mensagem para
você mesmo com o app aberto. O reenvio refaz a sessão; não é preciso parear de novo.

## 2. Manter sincronizado (daemon)
```sh
scripts/omni-wa-agent.sh install   # LaunchAgent do macOS: sobe no login e reinicia se cair
scripts/omni-wa-agent.sh status    # dono da sessão, cotas das últimas 24h, fila de áudios
scripts/omni-wa-agent.sh logs
scripts/omni-wa-agent.sh uninstall # obrigatório antes de omni-wa sync/logout
```
O daemon é o **único processo conectado**. Ele grava mensagens novas e arquivamentos, baixa e transcreve
áudios novos (whisper.cpp + ffmpeg; veja `WHISPER_BIN`, `WHISPER_MODEL`, `WHISPER_LANG` e `FFMPEG_BIN` em
`internal/connectors/whatsapp/transcribe.go`) e atende os envios do operador por um socket local.

**Modo seguro:**
- toda ação passa por cotas por hora e por dia, com intervalo aleatório e, para trabalho de fundo, só das 9h às 21h (`guard.go`);
- trabalho sobre mensagens antigas (histórico sob demanda, reenvio de mídia expirada) vem desligado; `OMNI_WA_BACKLOG=on` no `install` liga, devagar;
- `OMNI_WA_SEND=off` no `install` desliga o envio.

Outros comandos: `omni-wa chats`, `omni-wa export -archived -groups -out <pasta>`, `omni-wa transcribe`,
`omni-wa logout` e `omni-wa reset` (apaga localmente um aparelho que o celular já desconectou).

## 3. Usar no Claude Code
```sh
scripts/omni-wa-agent.sh claude
```
O comando faz três coisas:
- instala o `omni-mcp`;
- registra o servidor MCP `omni` no escopo *user*, com `OMNI_WA_READONLY=on`: as sessões do Claude só leem o banco e enviam pelo daemon, nunca abrem conexão própria;
- copia a skill [`whatsapp-contexto`](../OS-Optimizer-Playbook/skills/whatsapp-contexto/SKILL.md) para `~/.claude/skills/`.

| Ferramenta | Para quê |
|---|---|
| `whatsapp_recent_activity` | Tudo das últimas N horas, agrupado por chat |
| `whatsapp_search_messages` | Busca por termos no texto e nos áudios transcritos |
| `whatsapp_get_recent_messages` | Últimas mensagens de um chat |
| `whatsapp_search_chats` / `whatsapp_list_chats` | Localizar chats e grupos |
| `whatsapp_get_connection_status` | Daemon ativo? Última mensagem gravada |
| `whatsapp_send_message` | Envio pelo operador |

**Regras do operador** (`operator.go`):
- só envia para chats em que a pessoa já escreveu;
- até 10 por hora e 40 por dia, das 9h às 21h;
- mostra "digitando…" antes de enviar;
- recusa o mesmo texto para vários chats.

A skill orienta o Claude a mostrar o texto e esperar sua confirmação antes de cada envio.
