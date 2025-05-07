## 🌿 Agent Git Workflow & Concurrency Model

This spec describes how Guild agents can operate concurrently using Git branches as isolated sandboxes, while collaborating through shared interfaces and automated merges. The system is designed to support concurrent development with minimal merge conflicts and tight coordination via Kanban state tracking.

⸻

🧠 Goals
• Allow each agent to work on tasks in parallel without overwriting each other
• Use Git branches for sandboxing agent outputs
• Encourage shared interface definition while isolating implementation
• Prevent agents from working on incomplete or changing interfaces
• Enable the manager to orchestrate merges, resolve conflicts, and unblock dependent work

⸻

## 🌱 Git Workflow per Agent

Initialization: 1. Manager creates a repo and initializes a default main branch 2. Each agent creates or is assigned its own working branch: agent/<name> 3. Manager checks out each branch before assigning tasks

Agent Behavior:
• Agents always work in their own branch
• Tools and code assistants operate within this branch and working directory
• When a task is complete:
• Commit with message: feat: complete <task-id>
• Push to remote
• Notify manager via ZeroMQ

Manager Behavior:
• Subscribes to all agent/\* branches
• Periodically attempts to merge into main
• May run aider or another assistant to resolve merge conflicts
• Annotates the Kanban board with merge status

⸻

## 🧰 Agent Sandboxing

Per-Agent Environment:
• Each agent executes in its own:
• Branch (agent/<name>)
• Temporary working directory (/tmp/guild/<agent>/)
• Task scope limited to objective subtree (e.g. objectives/backend/)
• Optional: Mount only relevant portions of the repo per agent
• Optional: Use file system access policies (e.g. with gvisor, bwrap, or Docker)

⸻

## 🤝 Interface Blocking Mechanism

Interfaces — such as APIs, protobuf schemas, or shared models — are special.

Behavior: 1. A task is tagged interface 2. Any task that depends on this interface is marked Blocked in the Kanban board until it is Done 3. When the interface is updated:
• Dependent tasks are moved back to Blocked
• Agent memory is cleared or refreshed via RAG

Example:
• backend/api/user.go defines user endpoint
• frontend/pages/login.tsx depends on it
• While user.go is In Progress, login.tsx is Blocked
• When user.go is completed and committed:
• login.tsx unblocks
• PromptChain injects final interface as context

⸻

## 🔄 Continuous Merge Awareness

Agents working in parallel can periodically merge the latest main into their branch:

git fetch origin
git merge origin/main

Or: Manager pushes interface snapshots into a shared interface branch (interfaces/) which all agents subscribe to.

Agents are encouraged to:
• Regularly rebase against the latest main
• Log or RAG-index changes to shared boundaries

⸻

## ✅ Summary

This workflow enables:
• Safe concurrent agent execution
• Shared progress through Git + Kanban coordination
• Interface-first development planning
• Clean human-in-the-loop merge control

## Next Steps

• Implement per-agent branch creation in runtime
• Add Kanban tagging for interface + dependency resolution
• Extend ZeroMQ spec to carry merge + interface events
