# Second Brain & AI OS — Universal Routing (Claude Engine)

See [[GEMINI.md]] for the full 3-layer architecture, navigation routing table, and operational rules.

## Operating Principles
- **Root Instructions:** Keep concise (<200 lines). Routing table directs to layer notes.
- **Context Layer:** `Context/` holds static facts (identity, company, ICP, team, tech stack).
- **Memory Layer:** `Projects/` (active work), `Meetings/` (call notes/transcripts), `Decisions/` (ADRs), `Daily/` (logs).
- **Action Layer:** `skills/` holds repeatable procedures.

## Quick Routing
| Domain | Path |
|---|---|
| Identity & Voice | `Context/identity.md` |
| Business & Offers | `Context/company.md` |
| ICP & Audience | `Context/icp.md` |
| Team & Roles | `Context/team.md` |
| Tech Stack & Tools | `Context/tech-stack.md` |
| Active Projects | `Projects/active-projects.md` |
| Meeting Records | `Meetings/` |
| Settled Decisions | `Decisions/` |
| Vault Optimizer Skill | `skills/optimizer/SKILL.md` |
| Meeting Intelligence Skill | `skills/meeting-intelligence/SKILL.md` |
| WhatsApp Context Skill | `skills/whatsapp-contexto/SKILL.md` |
