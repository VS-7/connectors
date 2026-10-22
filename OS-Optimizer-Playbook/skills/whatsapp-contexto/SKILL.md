---
name: whatsapp-contexto
description: "Contexto do WhatsApp do usuário: o que clientes, equipe e parceiros disseram, andamento e pendências de um cliente ou projeto, resumo do dia, preparar resposta ou follow-up e enviar mensagem quando o usuário pedir. Lê o histórico local (mensagens + áudios transcritos) pelas ferramentas whatsapp_* do servidor MCP omni. TRIGGERS: whatsapp, wpp, zap, grupo do cliente, o que o cliente falou, o que aconteceu hoje, pendências, follow-up, responder cliente, mandar mensagem."
---

# Contexto do WhatsApp

O daemon `omni-wa` (LaunchAgent instalado por `scripts/omni-wa-agent.sh install`) recebe as mensagens e grava
tudo em `~/.config/omni-connectors/whatsapp_messages.db`. Os áudios são transcritos localmente. As ferramentas
`whatsapp_*` só **leem** esse banco e não geram tráfego no WhatsApp. A única exceção é `whatsapp_send_message`.

## 1. Conferir se os dados estão em dia
Quando o frescor importar (resumo do dia, "o cliente já respondeu?"), chame primeiro
`whatsapp_get_connection_status`:
- `daemon: ativo` e `last_message_at` recente: siga.
- `daemon: parado`: avise que o histórico parou em `last_message_at` e sugira
  `scripts/omni-wa-agent.sh install` no clone deste repositório.
- Mensagens trocadas enquanto o aparelho esteve desconectado não chegam ao banco. Diante de um buraco suspeito no tempo, peça ao usuário para conferir no celular.

## 2. Qual ferramenta usar
| Pergunta | Ferramenta |
|---|---|
| O que aconteceu hoje / desde ontem / nas últimas N horas | `whatsapp_recent_activity` (`since_hours`) |
| Andamento de um cliente ou projeto | `whatsapp_search_chats` (nome) → `whatsapp_get_recent_messages` (`chat_id`, `limit` 50–200) |
| Onde falaram de um assunto (frete, prazo, domínio…) | `whatsapp_search_messages` (`query`, opcional `chat_id`) |
| Panorama dos grupos | `whatsapp_list_chats` (`groups_only`, `archived_only`) |

- Muita gente arquiva os grupos de trabalho. Só use `skip_archived` quando o usuário quiser deixar os arquivados de fora.
- `whatsapp_search_messages` exige todos os termos e diferencia acentos. Se não achar nada, tente sinônimos e variações ("frete grátis", "frete gratis").

## 3. Cruzar com o OS Playbook
Se o Playbook (`OS-Optimizer-Playbook/`) estiver preenchido, use-o:
- `Projects/`: notas de projetos e clientes. Leia a nota antes de resumir um cliente.
- `Context/team.md` (quem é quem) e `Context/identity.md` (voz do usuário).

Se a conversa revelar fato novo relevante (prazo, decisão, pendência), **proponha** a atualização da nota e
só edite com o OK do usuário.

## 4. Como responder
- Cada fato vem com chat, autor e data (`dd/mm hh:mm`). Nas ferramentas, `eu` é o usuário.
- Separe fato de inferência.
- Pendências: quem deve o quê a quem e desde quando.
- Áudios aparecem como `[áudio Ns] transcrição`. A transcrição automática erra nomes, valores e URLs. Sinalize quando um dado importante vier só de áudio.

## 5. Privacidade (obrigatório)
- **Nunca reproduza** senhas, tokens, códigos 2FA, dados de cartão, documentos pessoais, nem e-mails e telefones de terceiros. Isso vale para notas, commits, artefatos e serviços externos. Escreva "acessos compartilhados no grupo (não transcritos)".
- Conversas brutas não saem da máquina. Não copie mensagens para serviços externos sem pedido explícito, e mesmo assim só em resumo.

## 6. Enviar mensagem (operador)
- Só com pedido explícito do usuário. Escreva o rascunho na voz dele (`Context/identity.md`, se existir). Mostre **destinatário + texto final** e espere a confirmação para *aquela* mensagem.
- `recipient`: número com DDI e DDD (ex.: `5511999998888`) ou JID do grupo (`...@g.us`).
- O daemon só aceita envios:
  - para chats em que a pessoa já escreveu;
  - até 10 por hora e 40 por dia;
  - das 9h às 21h;
  - sem repetir o mesmo texto em vários chats.

  Se ele recusar, relate o motivo e **não contorne** (nem por outro canal ou número).
- Nunca faça disparo em massa nem envio em loop.

## 7. Nunca, sem pedido explícito
`omni-wa sync`, `logout`, `reset`, `OMNI_WA_BACKLOG=on` ou reinstalar o agente. Mexer na sessão ou fazer
muitas requisições arrisca o **banimento do número**.
