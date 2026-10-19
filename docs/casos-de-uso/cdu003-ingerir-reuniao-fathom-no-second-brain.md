# CDU003 — Ingerir Reunião do Fathom no Second Brain

- **Atores:** Agente Second Brain / Operador
- **Componentes:** node-conn-fathom, node-ext-fathom, node-second-brain, node-core-hub

- **Complexidade:** high
- **Horas Estimadas:** 24
- **Prioridade:** Alta
- **Status:** implemented
- **Requisitos:** RF004, RF010, RNF002

## Descrição

O conector Fathom busca gravações recentes, transcrições com falantes e resumos gerados pela IA do Fathom, disponibilizando os dados em DTOs enxutos para que a skill `meeting-intelligence` do `OS-Optimizer-Playbook` processe a ata, extraia decisões e sincronize tarefas em `Projects/active-projects.md`.

## Pré-condições

- Variável `FATHOM_API_KEY` devidamente configurada no ambiente.
- O vault `OS-Optimizer-Playbook` acessível pelo agente no sistema de arquivos local.

## Pós-condições

- A ata da reunião é gerada no padrão `template-meeting.md` dentro de `OS-Optimizer-Playbook/Meetings/`.
- Decisões estratégicas e itens de ação são integrados nos projetos ativos do Second Brain.

## Fluxo Principal

1. O agente de IA executa a ferramenta `fathom_list_recordings` para identificar novas chamadas ou recebe o `recording_id` como parâmetro.
2. O conector Fathom faz a requisição autenticada à API externa do Fathom Video.
3. O conector processa a transcrição bruta, limpando metadados desnecessários e aplicando truncamento inteligente conforme o token budget (RNF002).
4. O conector retorna um DTO estruturado contendo: título, data, participantes, resumo executivo e segmentos de fala.
5. O agente aciona a skill `meeting-intelligence` do `OS-Optimizer-Playbook`.
6. A skill formata o documento da reunião em `Meetings/YYYY-MM-DD-titulo-da-reuniao.md`.
7. As tarefas levantadas com responsáveis e prazos são adicionadas à tabela de projetos em `Projects/active-projects.md`.

## Fluxos Alternativos

- **Indexação Local para Busca Rápida:** Caso configurado banco SQLite com FTS5, as transcrições são indexadas localmente para permitir buscas full-text instantâneas sem gastar chamadas repetidas à API.

## Exceções

- **Chave de API Inválida ou Ausente:** O conector retorna erro claro durante a inicialização (`FATHOM_API_KEY não configurada`), orientando o operador.
- **Gravação em Processamento:** Se o Fathom ainda estiver gerando a transcrição, a ferramenta responde com status indicativo de processamento pendente para nova tentativa posterior.

## Regras de Negócio

- Transcrições não devem sobrecarregar a janela de contexto; quando ultrapassarem 15.000 tokens, o conector deve priorizar o resumo estruturado e disponibilizar paginação ou busca por tópicos.
- Wikilinks do Second Brain (`[[active-projects]]`, `[[company]]`) devem ser preservados nas atas geradas.

## Critérios de Aceite

- **Given** uma gravação concluída no Fathom **When** o conector executa `fathom_get_call_transcript` **Then** retorna DTO contendo resumo e transcrição limpa sem HTML redundante.
- **Given** o retorno da ferramenta **When** o operador processa com `meeting-intelligence` **Then** um novo arquivo Markdown é criado em `OS-Optimizer-Playbook/Meetings/` e tarefas são cadastradas em `Projects/active-projects.md`.
