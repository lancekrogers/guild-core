# Simple API Development Task

## Objective
Create a basic REST API with essential endpoints to demonstrate Guild's code generation and testing capabilities.

## Requirements

### Core API Features
- Create a simple Go HTTP server
- Implement basic CRUD operations for a "tasks" resource
- Add proper error handling and HTTP status codes
- Include basic logging

### Technical Specifications
- Use Go's standard library (net/http)
- Implement JSON request/response handling
- Add input validation
- Follow REST conventions

### Endpoints Required
1. GET /tasks - List all tasks
2. POST /tasks - Create a new task  
3. GET /tasks/{id} - Get specific task
4. PUT /tasks/{id} - Update task
5. DELETE /tasks/{id} - Delete task

### Testing Requirements
- Write unit tests for each endpoint
- Include integration tests
- Test error scenarios
- Achieve >80% test coverage

## Success Criteria
- All endpoints respond correctly
- Tests pass and have good coverage
- Code follows Go best practices
- API is well-documented

## Notes
This is a demo commission designed to showcase Guild's multi-agent development workflow. The Manager will break this down into smaller tasks and assign them to appropriate specialized agents.