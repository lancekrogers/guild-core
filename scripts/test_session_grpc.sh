#!/bin/bash
# Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
# SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

# Guild gRPC Session Service Smoke Tests
# Tests the daemon session persistence API using grpcurl

set -euo pipefail

# Configuration
GRPC_HOST="${GRPC_HOST:-localhost:50051}"
GUILD_DAEMON_PID=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v grpcurl &> /dev/null; then
        log_error "grpcurl is required but not installed"
        log_info "Install with: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        log_info "Install with: brew install jq (macOS) or apt-get install jq (Linux)"
        exit 1
    fi
    
    log_success "All dependencies found"
}

# Start Guild daemon
start_daemon() {
    log_info "Starting Guild daemon..."
    
    # Build the daemon if needed
    if [[ ! -f "./bin/guild" ]]; then
        log_info "Building Guild binary..."
        make build || {
            log_error "Failed to build Guild binary"
            exit 1
        }
    fi
    
    # Start daemon in background
    ./bin/guild serve --port 50051 &
    GUILD_DAEMON_PID=$!
    
    # Wait for daemon to start
    log_info "Waiting for daemon to start..."
    for i in {1..30}; do
        if grpcurl -plaintext "$GRPC_HOST" list >/dev/null 2>&1; then
            log_success "Guild daemon started successfully (PID: $GUILD_DAEMON_PID)"
            return 0
        fi
        sleep 1
    done
    
    log_error "Failed to start Guild daemon"
    return 1
}

# Stop Guild daemon
stop_daemon() {
    if [[ -n "$GUILD_DAEMON_PID" ]]; then
        log_info "Stopping Guild daemon (PID: $GUILD_DAEMON_PID)..."
        kill "$GUILD_DAEMON_PID" 2>/dev/null || true
        wait "$GUILD_DAEMON_PID" 2>/dev/null || true
        log_success "Guild daemon stopped"
    fi
}

# Test gRPC reflection
test_reflection() {
    log_info "Testing gRPC reflection..."
    
    local services
    services=$(grpcurl -plaintext "$GRPC_HOST" list)
    
    if echo "$services" | grep -q "guild.v1.SessionService"; then
        log_success "SessionService found in reflection"
    else
        log_error "SessionService not found in reflection"
        return 1
    fi
    
    # Test method listing
    local methods
    methods=$(grpcurl -plaintext "$GRPC_HOST" list guild.v1.SessionService)
    
    local expected_methods=(
        "CreateSession"
        "GetSession"
        "ListSessions"
        "UpdateSession"
        "DeleteSession"
        "SaveMessage"
        "GetMessage"
        "GetMessages"
        "StreamMessages"
    )
    
    for method in "${expected_methods[@]}"; do
        if echo "$methods" | grep -q "$method"; then
            log_success "Method $method found"
        else
            log_error "Method $method not found"
            return 1
        fi
    done
}

# Test health check
test_health() {
    log_info "Testing health check..."
    
    local health_response
    health_response=$(grpcurl -plaintext "$GRPC_HOST" grpc.health.v1.Health/Check)
    
    if echo "$health_response" | jq -r '.status' | grep -q "SERVING"; then
        log_success "Health check passed"
    else
        log_error "Health check failed: $health_response"
        return 1
    fi
}

# Test session lifecycle
test_session_lifecycle() {
    log_info "Testing session lifecycle..."
    
    # Create session
    log_info "Creating session..."
    local create_response
    create_response=$(grpcurl -plaintext -d '{
        "name": "smoke-test-session",
        "metadata": {
            "test": "true",
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
        }
    }' "$GRPC_HOST" guild.v1.SessionService/CreateSession)
    
    local session_id
    session_id=$(echo "$create_response" | jq -r '.id')
    
    if [[ "$session_id" != "null" && -n "$session_id" ]]; then
        log_success "Session created with ID: $session_id"
    else
        log_error "Failed to create session: $create_response"
        return 1
    fi
    
    # Get session
    log_info "Retrieving session..."
    local get_response
    get_response=$(grpcurl -plaintext -d "{\"id\": \"$session_id\"}" "$GRPC_HOST" guild.v1.SessionService/GetSession)
    
    local retrieved_name
    retrieved_name=$(echo "$get_response" | jq -r '.name')
    
    if [[ "$retrieved_name" == "smoke-test-session" ]]; then
        log_success "Session retrieved successfully"
    else
        log_error "Failed to retrieve session: $get_response"
        return 1
    fi
    
    # Save message
    log_info "Saving message..."
    local message_response
    message_response=$(grpcurl -plaintext -d "{
        \"message\": {
            \"sessionId\": \"$session_id\",
            \"role\": \"USER\",
            \"content\": \"Hello from smoke test\",
            \"metadata\": {\"test\": \"message\"}
        }
    }" "$GRPC_HOST" guild.v1.SessionService/SaveMessage)
    
    local message_id
    message_id=$(echo "$message_response" | jq -r '.messageId')
    
    if [[ "$message_id" != "null" && -n "$message_id" ]]; then
        log_success "Message saved with ID: $message_id"
    else
        log_error "Failed to save message: $message_response"
        return 1
    fi
    
    # Get messages
    log_info "Retrieving messages..."
    local messages_response
    messages_response=$(grpcurl -plaintext -d "{\"sessionId\": \"$session_id\"}" "$GRPC_HOST" guild.v1.SessionService/GetMessages)
    
    local message_count
    message_count=$(echo "$messages_response" | jq '.messages | length')
    
    if [[ "$message_count" -eq 1 ]]; then
        log_success "Messages retrieved successfully (count: $message_count)"
    else
        log_error "Unexpected message count: $message_count"
        return 1
    fi
    
    # List sessions
    log_info "Listing sessions..."
    local list_response
    list_response=$(grpcurl -plaintext -d '{"limit": 10}' "$GRPC_HOST" guild.v1.SessionService/ListSessions)
    
    local session_count
    session_count=$(echo "$list_response" | jq '.sessions | length')
    
    if [[ "$session_count" -ge 1 ]]; then
        log_success "Sessions listed successfully (count: $session_count)"
    else
        log_error "No sessions found in list"
        return 1
    fi
    
    # Update session
    log_info "Updating session..."
    local update_response
    update_response=$(grpcurl -plaintext -d "{
        \"id\": \"$session_id\",
        \"name\": \"updated-smoke-test-session\",
        \"metadata\": {\"updated\": \"true\"}
    }" "$GRPC_HOST" guild.v1.SessionService/UpdateSession)
    
    local updated_name
    updated_name=$(echo "$update_response" | jq -r '.name')
    
    if [[ "$updated_name" == "updated-smoke-test-session" ]]; then
        log_success "Session updated successfully"
    else
        log_error "Failed to update session: $update_response"
        return 1
    fi
    
    # Delete session
    log_info "Deleting session..."
    local delete_response
    delete_response=$(grpcurl -plaintext -d "{\"id\": \"$session_id\"}" "$GRPC_HOST" guild.v1.SessionService/DeleteSession)
    
    local delete_success
    delete_success=$(echo "$delete_response" | jq -r '.success')
    
    if [[ "$delete_success" == "true" ]]; then
        log_success "Session deleted successfully"
    else
        log_error "Failed to delete session: $delete_response"
        return 1
    fi
}

# Test error handling
test_error_handling() {
    log_info "Testing error handling..."
    
    # Test invalid session ID
    log_info "Testing invalid session ID..."
    local error_response
    if error_response=$(grpcurl -plaintext -d '{"id": "invalid-id"}' "$GRPC_HOST" guild.v1.SessionService/GetSession 2>&1); then
        log_error "Expected error for invalid session ID, but got success: $error_response"
        return 1
    else
        if echo "$error_response" | grep -q "NotFound\|not found"; then
            log_success "Properly handled invalid session ID"
        else
            log_error "Unexpected error for invalid session ID: $error_response"
            return 1
        fi
    fi
    
    # Test empty session name
    log_info "Testing empty session name..."
    if error_response=$(grpcurl -plaintext -d '{"name": ""}' "$GRPC_HOST" guild.v1.SessionService/CreateSession 2>&1); then
        log_error "Expected error for empty session name, but got success: $error_response"
        return 1
    else
        if echo "$error_response" | grep -q "InvalidArgument\|cannot be empty"; then
            log_success "Properly handled empty session name"
        else
            log_error "Unexpected error for empty session name: $error_response"
            return 1
        fi
    fi
}

# Test performance
test_performance() {
    log_info "Testing performance..."
    
    local start_time
    start_time=$(date +%s%N)
    
    # Create 10 sessions quickly
    local session_ids=()
    for i in {1..10}; do
        local response
        response=$(grpcurl -plaintext -d "{\"name\": \"perf-test-$i\"}" "$GRPC_HOST" guild.v1.SessionService/CreateSession)
        local session_id
        session_id=$(echo "$response" | jq -r '.id')
        session_ids+=("$session_id")
    done
    
    local end_time
    end_time=$(date +%s%N)
    local duration_ms
    duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    log_success "Created 10 sessions in ${duration_ms}ms (avg: $((duration_ms / 10))ms per session)"
    
    # Clean up
    for session_id in "${session_ids[@]}"; do
        grpcurl -plaintext -d "{\"id\": \"$session_id\"}" "$GRPC_HOST" guild.v1.SessionService/DeleteSession >/dev/null
    done
    
    if [[ $((duration_ms / 10)) -le 150 ]]; then
        log_success "Performance target met (≤150ms per session)"
    else
        log_warning "Performance target missed (>150ms per session)"
    fi
}

# Main test runner
run_tests() {
    log_info "Starting Guild gRPC Session Service smoke tests..."
    
    local tests=(
        "test_reflection"
        "test_health"
        "test_session_lifecycle"
        "test_error_handling"
        "test_performance"
    )
    
    local passed=0
    local failed=0
    
    for test in "${tests[@]}"; do
        log_info "Running $test..."
        if $test; then
            ((passed++))
        else
            ((failed++))
            log_error "Test $test failed"
        fi
        echo
    done
    
    log_info "Test Summary:"
    log_success "Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log_error "Failed: $failed"
        return 1
    else
        log_success "All tests passed!"
        return 0
    fi
}

# Cleanup on exit
cleanup() {
    stop_daemon
}

# Main execution
main() {
    trap cleanup EXIT
    
    check_dependencies
    start_daemon
    run_tests
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi