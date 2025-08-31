# Docker Testing Module for Guild Framework
# Import with: mod docker 'just/docker.just'

# Set working directory for module recipes
set working-directory := ".."

# Docker image names
test_image := "guild-test:latest"
user_image := "guild-user:latest"

# ============================================================================
# User Experience Testing
# ============================================================================

# Build the user experience Docker image
build-user:
    @echo "🏗️ Building user experience image..."
    docker build -f Dockerfile.user -t {{user_image}} .

# Start interactive shell as a regular user with Guild installed
shell: build-user
    @echo "🐚 Starting interactive shell..."
    @echo "💡 Try: guild init, guild serve, guild chat"
    docker run --rm -it \
        -v guild-user-home:/home/developer \
        -v guild-global:/home/developer/.guild \
        {{user_image}}

# Test guild init workflow
init: build-user
    @echo "🏰 Testing guild init..."
    docker run --rm {{user_image}} bash -c \
        "cd /home/developer/projects/web-app && \
         guild init --force && \
         echo '✅ Init successful' && \
         ls -la .campaign/"

# Test full workflow (init → serve → status)
workflow: build-user
    @echo "🔄 Testing full workflow..."
    docker run --rm {{user_image}} bash -c \
        "cd /home/developer/projects/web-app && \
         guild init --force && \
         guild serve --daemon & \
         sleep 2 && \
         guild status && \
         echo '✅ Workflow completed'"

# Test chat interface (requires daemon)
chat: build-user
    @echo "💬 Testing chat interface..."
    docker compose -f docker-compose.user.yml run --rm guild-user bash -c \
        "cd /home/developer/projects/web-app && \
         guild init --force && \
         echo 'Testing chat' | timeout 5 guild chat || true"

# ============================================================================
# Development Testing
# ============================================================================

# Build the test Docker image
build-test:
    @echo "🏗️ Building test image..."
    docker build -f Dockerfile.test -t {{test_image}} .

# Run unit tests in Docker
test: build-test
    @echo "🧪 Running unit tests..."
    docker run --rm \
        -v "$(pwd)/test-output:/test-output" \
        {{test_image}} \
        go test ./pkg/... ./internal/... ./tools/...

# Run integration tests in Docker
integration: build-test
    @echo "🔧 Running integration tests..."
    docker run --rm \
        -v "$(pwd)/test-output:/test-output" \
        {{test_image}} \
        go test -tags=integration ./integration/...

# Run specific package tests
test-pkg pkg: build-test
    @echo "📦 Testing package: {{pkg}}"
    docker run --rm {{test_image}} go test -v ./{{pkg}}/...

# Run benchmarks in Docker
bench: build-test
    @echo "⚡ Running benchmarks..."
    docker run --rm {{test_image}} \
        go test -bench=. -benchmem ./pkg/... | grep -E "Benchmark|ns/op"

# ============================================================================
# Composition Testing
# ============================================================================

# Start full Guild environment with docker-compose
up:
    @echo "🚀 Starting Guild environment..."
    docker compose -f docker-compose.user.yml up -d
    @echo "✅ Environment ready. Use 'just docker exec' to enter"

# Execute command in running container
exec cmd="bash":
    docker compose -f docker-compose.user.yml exec guild-user {{cmd}}

# View logs from containers
logs:
    docker compose -f docker-compose.user.yml logs -f

# Stop Guild environment
down:
    @echo "🛑 Stopping Guild environment..."
    docker compose -f docker-compose.user.yml down

# ============================================================================
# Cleanup
# ============================================================================

# Clean all Docker resources
clean:
    @echo "🧹 Cleaning Docker resources..."
    docker compose -f docker-compose.user.yml down -v 2>/dev/null || true
    docker volume rm guild-core_user-home 2>/dev/null || true
    docker volume rm guild-core_guild-global 2>/dev/null || true
    docker rmi {{test_image}} 2>/dev/null || true
    docker rmi {{user_image}} 2>/dev/null || true
    @echo "✅ Docker resources cleaned"

# Remove test output
clean-output:
    @echo "🗑️ Removing test output..."
    rm -rf test-output/
    rm -rf test-*.log

# ============================================================================
# Quick Commands
# ============================================================================

# Quick test a single command
quick cmd:
    @docker run --rm {{user_image}} {{cmd}}

# Check Guild version in container
version: build-user
    @docker run --rm {{user_image}} guild version

# Show container details
inspect:
    @echo "📋 User image details:"
    @docker images {{user_image}}
    @echo ""
    @echo "📦 Volumes:"
    @docker volume ls | grep guild || echo "No Guild volumes"

# ============================================================================
# CI/CD Commands
# ============================================================================

# Run CI pipeline tests
ci-test: build-test
    @echo "🔄 Running CI tests..."
    docker run --rm \
        -e CI=true \
        {{test_image}} \
        make ci-test

# Run CI integration tests
ci-integration: build-test
    @echo "🔄 Running CI integration tests..."
    docker run --rm \
        -e CI=true \
        {{test_image}} \
        make ci-integration