# ⚙️ Configuration

Guild uses YAML or JSON configuration files to define agents, tools, guilds, and model preferences. These can be extended with environment variables or CLI overrides.

---

## 📁 Config Layout

```yaml
agents:
  - name: planner
    model: claude-3-opus
    tools:
      - tree2scaffold
      - search-codebase

  - name: implementer
    model: ollama:gemma-2b
    tools:
      - make
      - aider

guilds:
  - name: guild-dev
    agents:
      - planner
      - implementer
```

- Supports loading from `guild.yaml`, `config.json`, or env path
- Agent config defines:

  - Name
  - Model
  - Tools
  - Personality (optional)
  - Prompt templates (optional)

---

## 💰 Cost Configuration

You can assign relative or absolute costs to tools and models:

```yaml
costs:
  cli_tools:
    default: 0

  local_models:
    gemma-2b: 1
    llama3-70b: 3

  api_models:
    openai-gpt-4: 60 # cost per 1M tokens
    claude-opus: 45
    deepseek-coder: 25
```

Used by:

- Agents during prompt planning
- Managers when balancing task assignments
- MCP to detect inefficiencies

---

## 🔐 API Key Management

- API keys can be defined in `.env` or loaded from config:

```yaml
api_keys:
  openai: "sk-..."
  anthropic: "claude-api-key"
```

- Values from `.env` or environment variables override file-based keys
- All sensitive keys should be excluded from version control
