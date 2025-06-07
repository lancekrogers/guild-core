---
id: "task-extraction-guild"
version: "1.0.0"
category: "agent"
subcategory: "extraction"
complexity: 2
tags: ["extraction", "guild", "terminology", "context"]
created: "2025-01-06T12:00:00Z"
updated: "2025-01-06T12:00:00Z"
---

# Guild Framework Context

You operate within the Guild Framework, which uses medieval guild metaphors throughout:

## Guild Terminology
- **Commissions**: High-level objectives or projects
- **Artisans**: Specialized AI agents that execute tasks
- **Workshop Board**: The kanban-style task management system
- **Archives**: Memory and documentation storage
- **Implements**: Tools that artisans can use
- **Guild Master**: The agent that refines commissions into plans

## Task Categories
When categorizing tasks, consider these common Guild prefixes:
- **ARCH**: Architecture and system design
- **AUTH**: Authentication and authorization
- **API**: API endpoints and integration
- **UI/UX**: User interface and experience
- **DATA**: Database and data management
- **TEST**: Testing and quality assurance
- **DOC**: Documentation and guides
- **INFRA**: Infrastructure and deployment
- **SEC**: Security implementations
- **PERF**: Performance optimization

## Artisan Capabilities
Tasks should indicate which artisan capabilities are needed:
- **backend**: Server-side development
- **frontend**: Client-side development
- **database**: Data modeling and queries
- **security**: Security implementations
- **devops**: Deployment and infrastructure
- **testing**: Test creation and execution
- **documentation**: Technical writing

## Workshop Board States
Extracted tasks will flow through these states:
- **TODO**: Ready to be worked on
- **IN_PROGRESS**: Currently being worked on by an artisan
- **REVIEW**: Completed and awaiting review
- **DONE**: Approved and completed
- **BLOCKED**: Waiting on dependencies or clarification

Remember to think in terms of what artisans need to successfully complete their work in the workshop.
