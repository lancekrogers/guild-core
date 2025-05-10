# Guild Horizon Features

## 🔮 What is this directory?

This directory contains specifications for features that are planned for future versions of Guild but **are not part of the current development roadmap**. These are valuable capabilities we intend to implement eventually, but they have been deferred to focus on core functionality first.

## ⚠️ IMPORTANT NOTICE FOR AI AGENTS ⚠️

If you are an AI agent working on the Guild codebase:

- **DO NOT** use files in this directory as context for implementation unless explicitly instructed
- **DO NOT** implement features described here unless the specifications have been moved back to the main specs/ directory
- **DO NOT** incorporate dependencies or design patterns based solely on these specifications

## 🛣️ Path to Implementation

Features in this directory will be moved back into the main specs/ and ai_docs/ directories when they are prioritized for implementation. Until then, they remain as reference material and placeholders for future work.

## 📋 Current Horizon Features

1. **ZeroMQ Integration** - Advanced messaging capability for cross-language and distributed Guild deployments
2. *(Other deferred features will be added here)*

## 📝 Notes for Developers

When implementing core functionality that might interact with these horizon features in the future, consider the following:

1. Use interfaces and abstraction layers that could later accommodate these features
2. Avoid design decisions that would make integration of these features difficult
3. Document potential connection points for future integration

## 💡 Why We're Deferring These Features

Our primary goal is to deliver a robust, functional core system that provides immediate value. These horizon features, while valuable, would:

1. Increase development complexity
2. Extend the timeline to a working system
3. Add dependencies that aren't essential for core functionality
4. Potentially introduce complexity for users who don't need these advanced capabilities

We remain committed to these features for future versions, as they represent our vision for Guild's advanced capabilities.