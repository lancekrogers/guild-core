# 🧪 Example Guilds & Use Cases

This document provides concrete examples of how Guild agents, tools, and objectives come together to complete projects across domains.

---

## 📨 Single-Agent Email Assistant

- Agent: `inbox-bot`
- Model: Claude 3 Sonnet
- Tooling: email parser, template rewriter
- Behavior:

  - Reads markdown spec describing inbox cleanup goal
  - Suggests email replies based on tagged objectives
  - Defers to user when unsure (Blocked)

---

## 🌐 Dev Guild: Go + HTMX Website Builder

- Agents:

  - `planner`: defines scaffolding & routes
  - `implementer`: writes Go/HTMX code
  - `reviewer`: inspects PR diffs

- Tools:

  - `tree2scaffold`, `aider`, `goreleaser`

- Outcome:

  - Fully generated MVP with build pipeline and docs
  - Uses CLI tools when applicable to reduce cost

---

## 🛒 Decentralized Marketplace

- Guild Objective: Build a blockchain-based luxury goods exchange
- Agents:

  - `manager`: coordination + MCP
  - `contract-dev`: solidity engineer
  - `frontend`: connects to wallet

- Tools:

  - `hardhat`, `foundry`, `vite`, `tailwind`

- Features:

  - Contracts versioned via Git
  - Repeat gas analysis tasks converted to `analyze-gas` CLI tool

---

## 💡 Custom Tool Optimization Workflow

- Manager agent detects 4+ high-cost GPT tasks doing markdown table reformatting
- Flags as `tool_candidate`
- User implements `format-md` CLI tool
- Tool is added to agent config
- Future tasks use CLI without prompting the LLM
