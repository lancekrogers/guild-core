# Guild System Overview

This document provides a high-level overview of the Guild framework and its components.

## The Guildhall: Core Architecture

The Guild framework is structured like a medieval guildhall - a central place where skilled artisans gather to practice their craft under organized governance. The system consists of several interconnected components:

1. **The GuildHall**: The central coordination system (Orchestrator)
2. **Guild Members**: LLM-powered artisans with different specializations (Agents)
3. **The Guild Charter**: Objectives and tasks to be accomplished
4. **The Guild Ledger**: Task tracking and workflow management (Kanban)
5. **The Guild Archives**: Storage for knowledge and memory (BoltDB/Vector stores)
6. **Guild Tools**: External capabilities that members can wield
7. **Guild Masters**: Special agents that coordinate other members

## Guild Member Types

1. **GuildMaster (Manager Agent)**
   - Oversees other guild members
   - Plans objectives into discrete tasks
   - Assigns work based on specialization
   - Monitors progress and provides direction
   - Manages budget and resource allocation

2. **Craftsman (Worker Agent)**
   - Executes specific tasks with precision
   - Utilizes specialized tools and knowledge
   - Reports progress back to GuildMaster
   - Adapts to changing requirements
   - Operates within cost budgets

3. **Specialist Crafters**
   - Guild members with unique capabilities
   - Handles specialized domains (coding, writing, research)
   - Often equipped with domain-specific tools

## Guild Communication

1. **The Guild Hall Messenger (ZeroMQ)**
   - Provides real-time notifications between members
   - Supports publish-subscribe messaging
   - Enables coordination across the guild

2. **Summons System (Event Bus)**
   - Task assignments and notifications
   - Status updates and completion events
   - Blocking issues and assistance requests

## Guild Archives (Memory System)

1. **Guild Chronicles (Chain Manager)**
   - Records conversation histories
   - Provides context for ongoing work
   - Preserves institutional knowledge

2. **Knowledge Repository (Vector Store)**
   - Semantic search capabilities
   - Retrieval Augmented Generation
   - Storage for resources and references

3. **Guild Ledgers (BoltDB)**
   - Persistent storage for guild operations
   - Task states and history
   - Member information and configurations

## Guild Tradecraft (Tools System)

1. **Standard Guild Tools**
   - File manipulation tools
   - Web research capabilities
   - Shell command execution

2. **Specialized Implements**
   - Code generation and analysis
   - Data processing utilities
   - Media handling tools

3. **Tool Registry**
   - Manages tool access and privileges
   - Tracks tool usage and costs
   - Provides tool documentation

## Cost Accounting System

1. **Cost Awareness**
   - Tracking of LLM API costs
   - Tool usage accounting
   - Budget enforcement
   
2. **Resource Optimization**
   - Model selection based on cost-effectiveness
   - Prompt optimization
   - Caching strategies

## Guild Operations

1. **Commissioning Process**
   - Objectives are translated into tasks
   - Resources are allocated
   - Guild members are assigned

2. **Crafting Cycle**
   - Tasks progress through states
   - Communication between members
   - Tool usage and knowledge access

3. **Quality Assurance**
   - Verification of completed work
   - Human oversight when needed
   - Refinement and iteration

## Implementation Technologies

1. **Core Technologies**
   - Go for concurrent, efficient operation
   - BoltDB for reliable storage
   - ZeroMQ for messaging

2. **AI Providers**
   - Support for major LLM providers (OpenAI, Anthropic, etc.)
   - Local model options (Ollama)
   - Cost-optimized provider selection

## Next Steps

For more detailed information, see:

- [guild_member_lifecycle.md](guild_member_lifecycle.md) - Detailed guild member lifecycle
- [guild_runtime.md](guild_runtime.md) - Runtime behavior of the guild system
- [task_execution_flow.md](task_execution_flow.md) - How tasks flow through the guild
- [objectives_ui.md](objectives_ui.md) - The user interface for defining objectives