---
name: convencoes-git
description: Padrões de branch, commit, pull request e tags do projeto, gerados de .arch/conventions.yaml. Use ao criar branch, escrever commit ou abrir PR.
category: padronizacao
version: 1.0.0
trigger: sempre
checks:
  - id: commits-na-convencao
    text: Todos os commits da branch seguem a convenção do projeto.
    command: archcode-studio git lint
    required: true
  - id: branch-na-convencao
    text: A branch segue o padrão de nomes do projeto.
    required: false
source: gerado
---

# Convenções de Git do projeto

## Quando usar

Sempre que criar branch, escrever commit ou abrir pull request. Os padrões vêm de `.arch/conventions.yaml` e são validados pelos hooks locais e por `archcode-studio git lint` na CI.

## Regras

1. **Branch:** `sprint-01/TASK-API-01-implementar-emissao-de-jwt` — padrão `sprint-{sprint}/{id}-{slug}`. Fora de sprint: `{kind}/{id}-{slug}`.
2. **Commit:** `Sprint 01 - Implementa emissão de JWT [TASK-API-01]` — padrão `Sprint {sprint} - {summary} [{id}]`. Fora de sprint: `Hotfix - Corrige expiração antecipada da sessão [BUG-007]`.
3. Assunto com até 72 caracteres, começando por um verbo no presente da 3ª pessoa (Implementa, Corrige, Refatora, Testa, Documenta, Configura…).
4. O id do item vai entre colchetes no fim do assunto e precisa existir em `.arch/plan/`. Commits só de planejamento usam o id da sprint (`[SPRINT-01]`) ou `[ARCH-PLAN]` — gere com `archcode-studio commit --plan`.
5. Corpo opcional explica o porquê; no fim, os trailers `Task:`, `Story:`, `Refs:` e `Co-Authored-By:` quando um agente participou. Use `propose_commit` (MCP) ou `archcode-studio commit` para gerar a mensagem.
6. **Pull request:** título `Sprint 01 - Emissão de JWT [TASK-API-01]`; um PR por tarefa; merge por squash.
7. **Tags:** sprint `sprint-01`, release `v1.0.0` — criadas só por humanos.
8. **Autonomia da IA:** Assistido — o agente propõe cada commit e um humano confirma; push e PR são humanos. Merge, tag e release são sempre humanos.

## Como verificar

- `commits-na-convencao`: `archcode-studio git lint` sem erros.
- `branch-na-convencao`: o nome da branch atual passa em `archcode-studio git lint --branch "$(git branch --show-current)"`.
