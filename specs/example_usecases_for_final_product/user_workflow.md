🧭 User Workflow Guide – Using Guild for Secure Agentic Execution

This guide walks through the typical workflow of using the Guild Framework to automate complex projects. It’s written from the perspective of a professional user — for example, a patent attorney building a private, local LLM assistant to draft patent specifications securely and cost-effectively.

Guild helps you manage a group of intelligent agents that:
• Break down your objectives
• Collaborate on structured tasks
• Use local tools and models where possible
• Preserve your data privacy

⸻

⚙️ Step 1: Define Your Objective in Markdown

Start by creating a /objectives/ directory that outlines your project as a tree of markdown files.

As a patent attorney, your root might look like:

```bash
/objectives
├── README.md # Overall workflow: draft → edit → format → file
├── drafting/
│ ├── intro-section.md # Goal: draft the abstract and summary
│ ├── claims.md # Goal: generate independent/dependent claims
├── tools/
│ └── claim-refiner.md # Describe custom tool for post-editing claims

```

Each markdown file is:
• Structured as a prompt
• Easy for agents and humans to read
• Linkable and tagged for RAG-based memory lookup

⸻

🤖 Step 2: Configure Secure Local Agents

Next, create a guild.yaml file that defines your agents:

```yaml
agents:
  - name: claim-drafter
    model: ollama:llama3-8b
    tools:
      - claim-refiner
      - aider

guilds:
  - name: patent-bot
    agents: [claim-drafter]
```

You’ll configure:
• Which local models to use (e.g. Ollama)
• Which tools are safe to run offline
• How much each model/tool “costs” from a system perspective

For attorneys, cost settings enforce offline-only execution:

```yaml
costs:
api_models:
default: 99999
local_models:
llama3-8b: 1
cli_tools:
default: 0

```

⸻

📋 Step 3: Generate and Manage a Secure Kanban

Guild parses your objective tree into a Kanban board per agent. Each task is:
• Assigned based on agent roles
• Stored in BoltDB (private)
• Updated live via ZeroMQ messages (optional)

You can run guild dashboard or subscribe to events to:
• Watch agents work
• See which tasks are blocked
• Monitor tool usage in real time

⸻

🛑 Step 4: Human-in-the-Loop for Legal Judgment

If an agent encounters ambiguity or needs judgment:
• It marks the task Blocked
• You are notified and prompted to review
• You can:
• Edit the spec
• Approve the tool output
• Add legal citations or confidential notes manually

Because no API calls are made, you can safely edit or approve content without leaking privileged data.

⸻

🧠 Step 5: Continuous Optimization via MCP + RAG

Guild automatically watches for cost or repetition patterns:
• If your claim-drafter agent uses too many tokens formatting claim sections, the manager flags it
• A tool_candidate is logged (e.g. “format-claims”)
• You (or a dev) can implement a CLI tool to automate that step next time

Agents:
• Always restore their context from previous tasks using RAG
• Include memory of tool usage in their reasoning
• Prefer local tools when available

⸻

✅ Final Result

You’ve deployed a privacy-preserving legal automation system:
• Every step is traceable, editable, and version-controlled
• No API calls = no data leaks
• Local models and CLI tools = low cost
• Agents self-optimize and escalate edge cases

You’ve now converted a sensitive legal process into a repeatable agentic workflow — without compromising security or quality.
