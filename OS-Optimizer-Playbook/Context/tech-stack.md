---
status: active
type: context
tags: [tech-stack, tools, integrations]
date: 2026-10-18
---

# Technology Stack & Tooling

## 1. AI & Agent Runtimes
- **Google Antigravity / Gemini CLI:** Desenvolvimento agêntico, subagents, background tasks e análise de repositórios com janela de 1M-2M tokens.
- **Claude / Claude Code:** Raciocínio, coding e automações locais.
- **Spark:** Prototipação rápida e micro-agentes de execução.

## 2. Knowledge & Second Brain Base
- **Markdown Vault (Obsidian / Local Git / Baalda):** Base local descentralizada e versionada.
- **Format:** Standard Markdown com `[[wikilinks]]` e frontmatter YAML.

## 3. Operations & Integrations
- **Universal Connectors Hub (Go):** Super-server local (`omni-mcp` via stdio/SSE e `omni-http` via REST) desacoplado em Arquitetura Hexagonal, fornecendo ferramentas agnósticas para LLMs e Second Brain.
  - **Fathom Video:** Transcrições e resumos estruturados alimentando `Meetings/` e `skills/meeting-intelligence/`.
  - **WhatsApp:** Conexão multidevice (whatsmeow / SQLite local) para alertas e leitura de mensagens.
  - **Slack & Mattermost:** Interação com canais corporativos e busca em conversas passadas.
  - **Google Workspace:** Busca e leitura de mensagens no Gmail e arquivos no Google Drive.
- **Meeting Recorders:** Fathom / tl:dv / Fireflies / Google Meet (transcrições automáticas enviadas para `Meetings/`).
- **Automação:** n8n / Make / Webhooks para integração contínua entre CRM, agenda e o vault.

