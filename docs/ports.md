# Guild Framework Port Allocation

This document tracks port allocations for the Guild Framework to prevent conflicts.

## Production Ports

| Port Range | Service | Description |
|------------|---------|-------------|
| 8000-8099 | Guild API | Main REST/gRPC API endpoints |
| 8100-8199 | Guild Chat | WebSocket and streaming endpoints |
| 8200-8299 | Guild Metrics | Prometheus and observability endpoints |

## Test Ports

| Port Range | Purpose | Description |
|------------|---------|-------------|
| 56000-56099 | Unit Tests | Reserved for unit test fixed ports |
| 56100-56199 | Integration Tests | Reserved for integration test fixed ports |
| 56200-56299 | E2E Tests | Reserved for end-to-end test fixed ports |
| 56300-56999 | General Testing | Available for any test needs |

## Guidelines

1. **Prefer Dynamic Ports**: Always use port 0 (OS-assigned) when possible
2. **Document Fixed Ports**: If you must use a fixed port, add it to this document
3. **Avoid Conflicts**: Check this document before choosing a fixed port
4. **CI Considerations**: Ensure any fixed ports work in CI environments

## Specific Allocations

| Port | Test/Service | Reason |
|------|--------------|--------|
| 56000 | gRPC Health Probe Tests | Legacy health check requires fixed port |
| 56001 | Provider Mock Server | Simulates external provider APIs |

## Port Selection Algorithm

When a fixed port is required:

1. Check this document for conflicts
2. Choose from the appropriate range above
3. Add your allocation to the "Specific Allocations" table
4. Update your test to handle "address already in use" gracefully (3 retries with exponential backoff)
