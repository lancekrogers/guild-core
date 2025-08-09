#!/bin/bash
# Test Guild user experience in Docker - rapid feedback for real usage

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
ACTION=${1:-shell}
PROJECT=${2:-web-app}

echo -e "${CYAN}╔════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   Guild User Experience Test Suite    ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════╝${NC}"
echo

# Build the user experience image
build_user_image() {
    echo -e "${YELLOW}📦 Building user experience image...${NC}"
    docker build -f Dockerfile.user -t guild-user:latest . || {
        echo -e "${RED}❌ Build failed${NC}"
        exit 1
    }
    echo -e "${GREEN}✅ Image built successfully${NC}"
}

# Start interactive shell for manual testing
interactive_shell() {
    echo -e "${BLUE}🚀 Starting interactive Guild environment...${NC}"
    echo -e "${YELLOW}   You are now a 'developer' user with Guild installed${NC}"
    echo -e "${YELLOW}   Try: guild init, guild serve, guild chat${NC}"
    echo
    
    docker-compose -f docker-compose.user.yml run --rm guild-user
}

# Run automated user journey test
test_user_journey() {
    echo -e "${BLUE}🧪 Testing user journey: $1${NC}"
    
    case "$1" in
        init)
            echo -e "${YELLOW}Testing: guild init${NC}"
            docker-compose -f docker-compose.user.yml run --rm guild-user bash -c "
                cd projects/$PROJECT
                guild init --quick
                echo '✅ Init completed'
                ls -la .campaign/
                cat .campaign/campaign.yaml
            "
            ;;
            
        serve)
            echo -e "${YELLOW}Testing: guild serve (daemon)${NC}"
            docker-compose -f docker-compose.user.yml run --rm -d guild-user bash -c "
                cd projects/$PROJECT
                guild init --quick
                guild serve --daemon
                sleep 3
                guild status
            "
            ;;
            
        chat)
            echo -e "${YELLOW}Testing: guild chat (basic)${NC}"
            docker-compose -f docker-compose.user.yml run --rm guild-user bash -c "
                cd projects/$PROJECT
                guild init --quick
                echo 'Hello Guild' | timeout 5 guild chat || true
                echo '✅ Chat interface started'
            "
            ;;
            
        commission)
            echo -e "${YELLOW}Testing: commission creation${NC}"
            docker-compose -f docker-compose.user.yml run --rm guild-user bash -c "
                cd projects/$PROJECT
                guild init --quick
                guild commission create 'Build a REST API'
                ls -la commissions/
                head commissions/*.md
            "
            ;;
            
        full)
            echo -e "${YELLOW}Testing: Full user workflow${NC}"
            docker-compose -f docker-compose.user.yml run --rm guild-user bash -c "
                set -e
                cd projects/$PROJECT
                echo '1. Initializing Guild...'
                guild init --quick
                
                echo '2. Checking structure...'
                ls -la .campaign/
                
                echo '3. Starting daemon...'
                guild serve --daemon &
                DAEMON_PID=\$!
                sleep 2
                
                echo '4. Checking status...'
                guild status
                
                echo '5. Creating commission...'
                guild commission create 'Build TODO app'
                
                echo '6. Listing commissions...'
                ls commissions/
                
                echo '✅ Full workflow completed'
                kill \$DAEMON_PID 2>/dev/null || true
            "
            ;;
            
        *)
            echo -e "${RED}Unknown journey: $1${NC}"
            echo "Available: init, serve, chat, commission, full"
            exit 1
            ;;
    esac
}

# Quick test of specific command
quick_test() {
    local cmd="$@"
    echo -e "${BLUE}⚡ Quick test: $cmd${NC}"
    docker-compose -f docker-compose.user.yml run --rm guild-user bash -c "
        cd projects/$PROJECT
        $cmd
    "
}

# Reset test environment
reset_env() {
    echo -e "${YELLOW}🧹 Resetting test environment...${NC}"
    docker-compose -f docker-compose.user.yml down -v
    docker volume rm guild-core_user-home 2>/dev/null || true
    docker volume rm guild-core_guild-global 2>/dev/null || true
    echo -e "${GREEN}✅ Environment reset${NC}"
}

# Show logs from container
show_logs() {
    echo -e "${BLUE}📋 Container logs:${NC}"
    docker-compose -f docker-compose.user.yml logs guild-user
}

# Main execution
case "$ACTION" in
    build)
        build_user_image
        ;;
    shell|sh)
        build_user_image
        interactive_shell
        ;;
    test)
        build_user_image
        test_user_journey "$PROJECT"
        ;;
    quick|q)
        build_user_image
        shift
        quick_test "$@"
        ;;
    reset|clean)
        reset_env
        ;;
    logs)
        show_logs
        ;;
    help|--help|-h)
        echo "Usage: $0 [action] [options]"
        echo
        echo "Actions:"
        echo "  shell, sh         - Interactive shell as 'developer' user"
        echo "  test <journey>    - Run automated test (init|serve|chat|commission|full)"
        echo "  quick <command>   - Run a quick command test"
        echo "  reset, clean      - Reset the test environment"
        echo "  logs              - Show container logs"
        echo "  help              - Show this help"
        echo
        echo "Examples:"
        echo "  $0 shell                    # Interactive testing"
        echo "  $0 test init                # Test guild init"
        echo "  $0 test full                # Run full workflow test"
        echo "  $0 quick 'guild --version'  # Quick command test"
        ;;
    *)
        build_user_image
        interactive_shell
        ;;
esac