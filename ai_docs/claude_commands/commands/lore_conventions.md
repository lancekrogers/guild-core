## Guild Lore and Naming Conventions

@context

This command ensures Claude Code understands and applies Guild's lore and naming conventions consistently throughout the codebase.

### Load Guild Lore

I will now load and review the Guild lore and naming conventions from `ai_docs/project_context/lore.md`:

```bash
cat ai_docs/project_context/lore.md
```

### Naming Conventions Summary

Guild's naming conventions are based on medieval guild terminology, reflecting the project's collaborative nature:

1. **Core Concepts**

   - **Guild**: A collection of agents working toward a shared objective
   - **Agent**: An autonomous worker with specific skills and responsibilities
   - **Artisan**: A specialized agent focused on creative or skilled tasks
   - **Manager**: An agent that coordinates other agents
   - **Apprentice**: A learning or training agent

2. **Task and Process Terms**

   - **Craft**: The process of creating content or code
   - **Workshop**: The environment where agents operate
   - **Journeyman's Task**: A moderate complexity task
   - **Master Work**: A complex, high-quality output

3. **Directory and Structure Names**
   - **Hall**: Central coordination space
   - **Chamber**: Specialized working area
   - **Archive**: Storage for completed work
   - **Ledger**: Record-keeping system

### Application of Lore in Codebase

When implementing code, ensure these naming conventions are followed:

1. **Package Names**

   - Use guild-related terminology for package names where appropriate
   - Maintain clarity while honoring the theme

2. **Type and Interface Names**

   - Name interfaces and types according to the guild metaphor
   - Examples:
     - `GuildMaster` rather than `Orchestrator`
     - `Craftsman` rather than `Worker`
     - `Apprentice` rather than `TraineeAgent`

3. **Method Names**

   - Use thematic verb phrases for methods
   - Examples:
     - `CraftSolution` rather than `Generate`
     - `InspectWork` rather than `Validate`
     - `ApprenticeToMaster` rather than `Promote`

4. **CLI Command Names**
   - Structure commands to follow guild terminology
   - Examples:
     - `guild establish` rather than `guild init`
     - `guild commission` rather than `guild start`
     - `guild inspect` rather than `guild status`

### Documentation Style

All documentation should:

1. Use guild-appropriate language and metaphors
2. Maintain medieval guild terminology consistently
3. Explain technical concepts using guild analogies
4. Include appropriate emoji icons as specified in the lore document

### UI Text and Messages

User interface elements should:

1. Reflect guild terminology in labels and buttons
2. Use period-appropriate language for confirmations and errors
3. Maintain the guild atmosphere throughout the user experience

When implementing any component of Guild, consistently apply these naming conventions and lore elements to create a cohesive, thematic experience.
