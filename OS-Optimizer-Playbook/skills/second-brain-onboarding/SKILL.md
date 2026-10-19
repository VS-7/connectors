---
name: second-brain-onboarding
description: "Interactive onboarding wizard to setup and populate a Second Brain / AI OS in 15 minutes. Interviews the operator to establish Context/ (identity, company, ICP, team, tech-stack) and initializes active projects. TRIGGERS: setup second brain, onboarding, configurar segundo cerebro, criar segundo cerebro, init brain."
---

# Second Brain Onboarding Wizard

Guides the operator through an interactive, low-friction setup process to establish the canonical Layer 1 (Context) and Layer 2 (Memory) of their Second Brain.

## Operating Flow

### Phase 1 — Identity & Operator Voice (`Context/identity.md`)
Ask the operator:
1. What is your name and primary role in the company?
2. What is your preferred communication style (e.g., direct & concise, highly strategic, casual, technical)?
3. What are your non-negotiable principles when delegating or automating tasks?

*Action:* Update `Context/identity.md` with concrete answers, keeping bullet points crisp.

---

### Phase 2 — Business Model & Core Offer (`Context/company.md`)
Ask the operator:
1. What is the name of your business and what core problem does it solve?
2. What is your main flagship offer / service, and how is it delivered?
3. What is the typical pricing or engagement structure?

*Action:* Update `Context/company.md` with clear bulleted value propositions.

---

### Phase 3 — Ideal Customer Profile (`Context/icp.md`)
Ask the operator:
1. Who is your target decision-maker (role, company size, industry)?
2. What acute pain points trigger them to look for your solution?
3. What are the 2–3 most frequent objections you hear during sales calls?

*Action:* Update `Context/icp.md` so the AI can tailor sales copy, outreach, and analysis accurately.

---

### Phase 4 — Team & Operations (`Context/team.md` & `Context/tech-stack.md`)
Ask the operator:
1. Who are the key people on your team, and what decisions can be made autonomously?
2. What tools, meeting recorders, or automation platforms (n8n, Make, CRM) are active?

*Action:* Update `Context/team.md` and `Context/tech-stack.md`.

---

### Phase 5 — Initialize Active Projects (`Projects/active-projects.md`)
Ask the operator:
1. What are your top 1–3 active priorities this week or sprint?
2. Who owns each priority, and what is the next tangible milestone?

*Action:* Populate `Projects/active-projects.md`.

---

### Phase 6 — Verification Run with OS Optimizer
Once Layer 1 and Layer 2 are initialized:
1. Prompt the operator: "Seu Segundo Cérebro está estruturado! Deseja rodar o OS Optimizer agora para validar os links e a arquitetura?"
2. When approved, invoke `skills/optimizer/`.
