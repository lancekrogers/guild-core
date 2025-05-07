# 💾 Memory, PromptChain, and Retrieval

The Guild framework includes structured memory systems to support context continuity, prompt reuse, and low-cost decision-making.

---

## 🧠 PromptChain

Each agent maintains a history of prompt–response pairs per task. This is referred to as a `PromptChain`.

- Stored in-memory during execution
- Archived to disk (BoltDB) upon task completion
- Indexed by task ID, agent ID, timestamp, and outcome
- Supports replay and visualization

Used for:

- Debugging
- Model introspection
- Prompt rehydration

---

## 📦 Vector Store (Qdrant)

Guild uses **Qdrant** for semantic memory and fast similarity search.

Indexed content includes:

- Past prompt chains
- Markdown objective files
- Final task outputs or summaries

Qdrant allows agents to:

- Retrieve similar past tasks
- Match new problems to existing context
- Inject known patterns into current reasoning

---

## 🔍 Retrieval-Augmented Generation (RAG)

Agents use RAG to:

- Load relevant context before executing a new prompt
- Pre-fill prompts with summaries instead of repeating reasoning
- Justify tool or model usage based on past outcomes

Sources for retrieval:

- Qdrant (semantic)
- BoltDB (symbolic or tag-based)
- File system (Markdown objectives)

---

## 🔄 Efficient Context Restoration

To reduce token usage and avoid re-generation:

- Context is loaded before every prompt
- Cost metadata and CLI tool info are included
- Rehydrated prompts carry references to tool usage

This mechanism enables agents to:

- Make cost-efficient decisions by default
- Avoid excessive API calls
- Learn from guild history across runs
