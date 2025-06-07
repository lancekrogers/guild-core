#!/usr/bin/env bash
#
# Proto Generation Script for Guild Framework
# This script generates Go code from Protocol Buffer definitions
#
# Prerequisites:
# - protoc (Protocol Buffer Compiler) v5.28.3 or compatible
# - protoc-gen-go v1.36.6 or compatible
# - protoc-gen-go-grpc (for gRPC service generation)
#
# Install prerequisites:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
# Usage:
#   ./scripts/generate-proto.sh

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

echo "=== Guild Framework Proto Generation ==="
echo "Project root: $PROJECT_ROOT"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Please install protoc from https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Error: protoc-gen-go is not installed"
    echo "Please run: go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6"
    exit 1
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Error: protoc-gen-go-grpc is not installed"
    echo "Please run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

echo "✓ All prerequisites found"
echo ""

# Function to generate proto files
generate_proto() {
    local proto_file=$1
    local output_dir=$2

    echo "Generating: $proto_file"
    echo "Output dir: $output_dir"

    # Create output directory if it doesn't exist
    mkdir -p "$output_dir"

    # Generate Go code
    protoc \
        --go_out="$PROJECT_ROOT" \
        --go-grpc_out="$PROJECT_ROOT" \
        --proto_path="$PROJECT_ROOT" \
        "$proto_file"

    echo "✓ Generated successfully"
    echo ""
}

# Generate Guild service proto
echo "=== Generating Guild Service ==="
generate_proto "proto/guild/v1/guild.proto" "pkg/grpc/pb/guild/v1"

# Generate Chat service proto
echo "=== Generating Chat Service ==="
generate_proto "proto/guild/v1/chat.proto" "pkg/grpc/pb/guild/v1"

# Generate MCP service proto
echo "=== Generating MCP Service ==="
generate_proto "proto/mcp/v1/mcp.proto" "pkg/grpc/pb/mcp/v1"

# Generate Prompts service proto
echo "=== Generating Prompts Service ==="
generate_proto "proto/prompts/v1/prompts.proto" "pkg/grpc/pb/prompts/v1"

echo "=== Proto generation complete ==="
echo ""
echo "Generated files:"
echo "  - pkg/grpc/pb/guild/v1/guild.pb.go"
echo "  - pkg/grpc/pb/guild/v1/guild_grpc.pb.go"
echo "  - pkg/grpc/pb/guild/v1/chat.pb.go"
echo "  - pkg/grpc/pb/guild/v1/chat_grpc.pb.go"
echo "  - pkg/grpc/pb/mcp/v1/mcp.pb.go"
echo "  - pkg/grpc/pb/mcp/v1/mcp_grpc.pb.go"
echo "  - pkg/grpc/pb/prompts/v1/prompts.pb.go"
echo "  - pkg/grpc/pb/prompts/v1/prompts_grpc.pb.go"
echo ""
echo "Note: These files should NOT be edited manually."
echo "Any changes should be made to the .proto files and regenerated."
