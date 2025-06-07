---
id: "task-extraction-domain"
version: "1.0.0"
category: "agent"
subcategory: "extraction"
complexity: 2
tags: ["extraction", "domain", "context", "variable"]
variables:
  required: ["DomainType", "DomainContext"]
created: "2025-01-06T12:00:00Z"
updated: "2025-01-06T12:00:00Z"
---

# Domain-Specific Understanding

You are extracting tasks for a {{.DomainType}} project. This domain has specific patterns and requirements:

## Domain Context
{{.DomainContext}}

## Common Task Patterns for {{.DomainType}}

Based on the domain type, prioritize and categorize tasks appropriately. Consider:

1. **Technical Requirements**: What are the key technical components?
2. **User Interactions**: How will users interact with this system?
3. **Data Flow**: How does data move through the system?
4. **Integration Points**: What external systems need to be connected?
5. **Quality Measures**: What testing and validation is needed?

## Priority Guidance

For {{.DomainType}} projects, consider these priority factors:
- **Critical Path**: What must be done first to unblock other work?
- **User Impact**: What affects the user experience most directly?
- **Technical Dependencies**: What foundational work enables other tasks?
- **Risk Mitigation**: What reduces project risk if done early?

## Estimation Considerations

When estimating task duration for {{.DomainType}}:
- Consider complexity relative to the domain
- Account for integration and testing time
- Include time for documentation and review
- Factor in any domain-specific requirements

Remember that different domains have different complexity patterns. A simple task in one domain might be complex in another.
