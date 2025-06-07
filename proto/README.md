# Protocol Buffer Definitions

This directory contains the Protocol Buffer (protobuf) definitions for the Guild Framework's gRPC services.

## Proto Files

- `guild/v1/guild.proto` - Core Guild service definitions
- `guild/v1/chat.proto` - Chat service for agent interactions
- `mcp/v1/mcp.proto` - Meta-Coordination Protocol service
- `prompts/v1/prompts.proto` - Prompt management service

## Generating Go Code

The Guild Framework provides multiple ways to generate Go code from these proto files:

### Using Taskfile (Recommended)
```bash
# Install required tools
task proto:install

# Generate Go code
task proto

# Verify proto files are valid
task proto:check
```

### Using Makefile
```bash
# Generate Go code
make proto

# Verify proto files are valid
make proto-check
```

### Using go generate
```bash
# From the project root
go generate ./...
```

### Manual generation
```bash
# Run the generation script directly
./scripts/generate-proto.sh
```

## Prerequisites

1. **protoc** - Protocol Buffer Compiler
   - Download from: https://github.com/protocolbuffers/protobuf/releases
   - Version: v5.28.3 or compatible

2. **Go protobuf plugins**
   - Install with: `task proto:install` or manually:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

## Generated Files

The generated Go files are placed in the following locations:

- `pkg/grpc/pb/guild/v1/` - Guild service
- `pkg/grpc/pb/` - Chat service
- `pkg/mcp/grpc/` - MCP service
- `pkg/grpc/pb/prompts/v1/` - Prompts service

## Important Notes

1. **DO NOT EDIT** generated files (`*.pb.go` and `*_grpc.pb.go`)
2. All changes should be made to the `.proto` files
3. Regenerate code after modifying proto files
4. Commit both proto files and generated code to version control

## Adding New Proto Files

1. Create the `.proto` file in the appropriate directory
2. Add the `go_package` option pointing to the correct output location
3. Update the generation script (`scripts/generate-proto.sh`) to include the new file
4. Add a new `//go:generate` directive in `tools.go`
5. Run the generation command

## Example Proto Definition

```protobuf
syntax = "proto3";

package guild.v1;

option go_package = "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1";

service GuildService {
  rpc CreateGuild(CreateGuildRequest) returns (CreateGuildResponse);
}

message CreateGuildRequest {
  string name = 1;
}

message CreateGuildResponse {
  string guild_id = 1;
}
```
