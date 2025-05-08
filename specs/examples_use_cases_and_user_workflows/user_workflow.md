# 🧭 Guild User Workflow Guide  
**Secure, Private, Agentic Automation for Professionals**

This guide walks you through using the Guild Framework to automate complex, high-trust workflows — for example, drafting patent specifications using local LLMs without sending any data to external APIs.

Guild lets you orchestrate intelligent agents that:
- Break down objectives into tasks
- Execute tasks collaboratively
- Use local models and tools
- Preserve privacy and cut costs

---

## ⚙️ Step 1: Define Your Objective as a Markdown Tree

Create an `/objectives/` directory. Each `.md` file is a structured, tagged prompt used by agents.

Example for a patent automation workflow:

```text
/objectives
├── README.md                  # Full process: draft → edit → format → file
├── drafting/
│   ├── intro-section.md       # Goal: Write abstract and summary
│   └── claims.md              # Goal: Generate independent and dependent claims
└── tools/
    └── claim-refiner.md       # Describe post-processing tool for polishing claims
```

Each file:
- Follows a standard prompt format (`Goal`, `Context`, `Requirements`, `Tags`)
- Is human-readable and machine-parseable
- Links to related tasks and tools

---

## 🤖 Step 2: Configure Local-First Agents

In `guild.yaml`, define your agents and guilds.

```yaml
agents:
  - name: claim-drafter
    model: ollama:llama3-8b
    tools: [claim-refiner, aider]

guilds:
  - name: patent-bot
    agents: [claim-drafter]

costs:
  api_models:
    default: 99999      # Block API usage
  local_models:
    llama3-8b: 1         # Encourage local inference
  cli_tools:
    default: 0           # Prefer shell tools
```

This setup:
- Forces local-only execution (no outbound API risk)
- Lets agents call CLI tools you define
- Assigns system cost weights to guide scheduling

---

## 📋 Step 3: Launch the Kanban-Based Task System

Run:

```bash
guild run
```

The manager agent:
- Parses the `/objectives/` tree
- Plans a task graph
- Assigns tasks to agents based on tools, roles, and status

Tasks are:
- Stored in **BoltDB**
- Updated live via **ZeroMQ** (optional)
- Viewable with:

```bash
guild monitor
guild monitor --agent claim-drafter
```

You’ll see:
- Task queues per agent
- Blocked tasks
- Real-time status

---

## 🛑 Step 4: Human-in-the-Loop Review

Agents escalate unclear or sensitive decisions by marking tasks `Blocked`.

When this happens:
- You’re notified in the dashboard
- You can inspect, edit, or approve output
- Optionally add confidential notes, references, or overrides

No network calls are made, so legal or proprietary content stays secure.

---

## 🧠 Step 5: Self-Optimization via MCP & RAG

Guild constantly improves based on usage patterns.

Examples:
- If a model repeatedly fumbles formatting claims, a “tool_candidate” is logged
- You or a dev can implement that tool (e.g., `format-claims`)
- Next time, the agent uses it automatically

Agents:
- Use **RAG** to load related memory before each task
- Log tool usage and success/failure
- Prefer local deterministic tools over high-cost model calls

---

## ✅ Final Result: Private, Repeatable, Auditable Automation

You now have:
- A reusable, traceable, agentic system for patent drafting
- Version-controlled, locally executed workflows
- Configurable guardrails (costs, tools, blocking)
- Full observability, no data leakage

With Guild, even complex, sensitive workflows become repeatable — without compromising security or quality.
