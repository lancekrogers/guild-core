# 🏛️ Guild Lore and Naming Conventions

This document outlines the conceptual foundation of the Guild Framework and provides canonical terms and naming guidelines. It draws inspiration from the historical tradition of medieval guilds — associations of skilled craftspeople and merchants who governed the practice, tools, and integrity of their trade.

---

## 🧱 Historical Inspiration

A **guild** was a self-regulating organization of artisans or merchants that:

- Oversaw quality and ethics of their trade
- Protected access to tools, knowledge, and materials
- Maintained guildhalls (shared workspaces and repositories)
- Enforced privilege: only guild members could practice the craft
- Had internal discipline — members who violated the rules could be fined, censured, or expelled

This framework reimagines that structure for agentic software.

---

## 🧠 Conceptual Mapping

| Guild Term        | Agentic Equivalent                                    |
| ----------------- | ----------------------------------------------------- |
| Guild             | A group of agents with shared goals                   |
| Guildhall         | The working directory / project corpus                |
| Artisan           | An individual agent (e.g., planner, coder)            |
| Apprentice        | A lower-cost or narrower-scope agent                  |
| Tool              | Local CLI tools or APIs usable by agents              |
| Privilege         | Access control to models, tasks, or directories       |
| Tradecraft        | The coded "skills" encoded in specs or tools          |
| Objective Charter | The `/objectives/` root definition                    |
| Letter of Patent  | Approval to initiate a new project (i.e. guild start) |

---

## 🗂️ Naming Conventions

- **Guilds**: named after domains or purposes  
  e.g. `infra-guild`, `patent-guild`, `frontend-guild`

- **Agents**: named as roles or archetypes  
  e.g. `scribe`, `reviewer`, `crafter`, `navigator`, `scribe-apprentice`

- **Specs**: reflect “commissions” given to the guild  
  e.g. `specs/features/claim_drafting.md`, `specs/bugs/overpriced_api.md`

- **Tools**: use verbs or professions  
  e.g. `refine`, `engrave`, `sift`, `aider`, `claim-refiner`

- **Directories**:
  - `/guildhall` (optional): structured persistent workspace
  - `/objectives`: charter of the guild's purpose and subgoals
  - `/ai_docs`: codified trade knowledge
  - `/specs`: active commissions, proposals, and internal tasks
  - `/.guild/`: privileges, memory logs, and command rituals

---

## 🧾 Purpose of Lore

This file is intended to help contributors:

- Use consistent terminology
- Stay aligned with the narrative metaphor
- Maintain internal coherence between technical and thematic layers

It is not required for execution — but essential for identity.
