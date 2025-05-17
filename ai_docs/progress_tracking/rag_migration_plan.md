# 🔄 RAG and Agent Migration Plan

## Overview

This document outlines the plan for migrating important business logic from previous implementations of the RAG (Retrieval-Augmented Generation) and Agent systems into the new cleaner architecture. The goal is to maintain the architectural improvements while incorporating valuable features from the old code.

## Migration Checklist

### 1. Cost Tracking System

- [x] Basic cost tracking interface (pkg/agent/cost.go exists)
- [x] Expand cost tracking to include multiple cost types (LLM, Tool, etc.)
- [x] Implement budget management with limits
- [x] Add cost estimation for LLM operations
- [x] Add cost recording with metadata
- [x] Implement cost-aware behavior in agents
- [x] Add cost reporting capabilities

### 2. Agent Memory Management

- [x] Verify existing memory chain management
- [x] Implement memory chain creation for tasks
- [x] Add context building from memory
- [x] Implement token tracking for memory
- [x] Add message categorization (system, user, assistant, tool)
- [x] Ensure memory persistence across agent runs

### 3. RAG Agent Wrapper

- [x] Create agent wrapper pattern
- [x] Implement delegate methods for GuildArtisan interface
- [x] Add prompt enhancement with RAG context
- [x] Implement query extraction from tasks
- [x] Add memory chain management for enhanced prompts
- [x] Ensure proper interface with vector stores

### 4. Tool Registry and Execution

- [x] Expand tool registry implementation
- [x] Add tool execution with parameters
- [x] Implement tool result handling
- [x] Add cost tracking for tool usage
- [x] Implement tool-specific cost estimation
- [x] Add tool usage reporting

## Implementation Notes

### Cost Tracking System

The cost tracking system should manage different types of costs (LLM, tool, storage) for each agent. It should track usage, enforce budgets, and provide reports. The system should be integrated with the agent execution loop to make cost-aware decisions.

Key components:
- CostManager
- CostRecord
- CostType enums
- Budget enforcement
- Cost reporting

### Agent Memory Management

Memory management should handle the creation, update, and retrieval of memory chains for agent conversations. It should track token usage, categorize messages, and build context for LLM prompts.

Key components:
- Memory chain creation
- Message categorization
- Context building
- Token tracking

### RAG Agent Wrapper

The RAG agent wrapper should enhance existing agents with retrieval capabilities. It should intercept prompt construction to add relevant context from vector stores and manage memory chains for enhanced prompts.

Key components:
- Agent wrapper implementation
- Delegate methods for GuildArtisan interface
- Prompt enhancement with RAG context
- Query extraction from tasks

### Tool Registry and Execution

The tool registry should manage available tools, handle execution with parameters, and track costs. It should provide a unified interface for tool usage and reporting.

Key components:
- Tool registry expansion
- Tool execution with parameters
- Tool result handling
- Cost tracking for tools

## Conclusion

This migration plan outlines the key components to be migrated from the old implementation to the new architecture. By following this plan, we'll ensure that valuable business logic is preserved while maintaining the clean architecture of the new system.