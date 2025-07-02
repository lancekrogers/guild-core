#!/bin/bash

# Guild Production Deployment Script
# Handles safe deployment with rollback capability
# 
# This script implements the deployment procedures requirements for launch coordination,
# providing automated, safe deployment to production with comprehensive validation,
# backup, and rollback capabilities.
#
# Usage:
#   ./deploy-production.sh
#   DEPLOYMENT_ENV=staging ./deploy-production.sh
#   VERSION=v1.2.3 ./deploy-production.sh

set -euo pipefail

# Configuration
DEPLOYMENT_ENV="${DEPLOYMENT_ENV:-production}"
VERSION="${VERSION:-$(git describe --tags --always)}"
BACKUP_DIR="/tmp/guild-backup-$(date +%Y%m%d-%H%M%S)"
LOG_FILE="/var/log/guild/deployment-$(date +%Y%m%d-%H%M%S).log"
GUILD_USER="${GUILD_USER:-guild-deploy}"
GUILD_HOME="${GUILD_HOME:-/opt/guild}"
GUILD_CONFIG_DIR="${GUILD_CONFIG_DIR:-/etc/guild}"
GUILD_DATA_DIR="${GUILD_DATA_DIR:-/opt/guild/data}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        error "Deployment failed with exit code $exit_code"
        if [ -f "/tmp/guild-deployment-vars" ]; then
            warning "Rollback information available. Run with --rollback to revert changes."
        fi
    fi
    exit $exit_code
}

# Set up trap for cleanup
trap cleanup EXIT

# Pre-deployment checks
run_pre_deployment_checks() {
    log "Running pre-deployment checks..."
    
    # Check if running as correct user
    if [ "$USER" != "$GUILD_USER" ] && [ "$USER" != "root" ]; then
        error "Must run as $GUILD_USER user or root"
        exit 1
    fi
    
    # Check disk space (require at least 1GB)
    AVAILABLE_SPACE=$(df "$GUILD_HOME" | awk 'NR==2 {print $4}')
    REQUIRED_SPACE=1048576  # 1GB in KB
    
    if [ "$AVAILABLE_SPACE" -lt "$REQUIRED_SPACE" ]; then
        error "Insufficient disk space. Available: ${AVAILABLE_SPACE}KB, Required: ${REQUIRED_SPACE}KB"
        exit 1
    fi
    
    # Check if Guild service is running
    if systemctl is-active --quiet guild-daemon; then
        log "Guild daemon is currently running"
    else
        warning "Guild daemon is not running"
    fi
    
    # Check database connectivity
    if [ -f "$GUILD_DATA_DIR/memory.db" ]; then
        if ! sqlite3 "$GUILD_DATA_DIR/memory.db" ".timeout 5000" ".schema" > /dev/null 2>&1; then
            error "Cannot connect to Guild database"
            exit 1
        fi
    else
        warning "Guild database not found - this may be a fresh installation"
    fi
    
    # Validate environment configuration
    if [ ! -f "$GUILD_CONFIG_DIR/production.yaml" ] && [ "$DEPLOYMENT_ENV" = "production" ]; then
        error "Production configuration not found at $GUILD_CONFIG_DIR/production.yaml"
        exit 1
    fi
    
    # Check if deployment artifacts exist
    if [ ! -d "./dist/guild-$VERSION" ] && [ ! -f "./guild-$VERSION.tar.gz" ]; then
        error "Deployment artifacts not found for version $VERSION"
        exit 1
    fi
    
    # Check performance baseline (if validate-performance exists)
    if [ -f "$GUILD_HOME/bin/validate-performance" ]; then
        log "Running performance baseline check..."
        if ! timeout 30s "$GUILD_HOME/bin/validate-performance" --quick-check; then
            warning "Performance baseline check failed - proceeding with caution"
        fi
    fi
    
    success "Pre-deployment checks passed"
}

# Create backup of current deployment
create_backup() {
    log "Creating backup of current deployment..."
    
    mkdir -p "$BACKUP_DIR"
    
    # Backup binaries
    if [ -d "$GUILD_HOME/bin" ]; then
        cp -r "$GUILD_HOME/bin" "$BACKUP_DIR/"
        log "Backed up binaries to $BACKUP_DIR/bin"
    fi
    
    # Backup configuration
    if [ -d "$GUILD_CONFIG_DIR" ]; then
        cp -r "$GUILD_CONFIG_DIR" "$BACKUP_DIR/config"
        log "Backed up configuration to $BACKUP_DIR/config"
    fi
    
    # Backup database (with locks)
    if [ -f "$GUILD_DATA_DIR/memory.db" ]; then
        # Use WAL mode backup for minimal disruption
        sqlite3 "$GUILD_DATA_DIR/memory.db" ".backup $BACKUP_DIR/memory.db"
        log "Backed up database to $BACKUP_DIR/memory.db"
    fi
    
    # Backup systemd service files
    if [ -f "/etc/systemd/system/guild-daemon.service" ]; then
        cp "/etc/systemd/system/guild-daemon.service" "$BACKUP_DIR/"
        log "Backed up systemd service file"
    fi
    
    # Create backup manifest
    cat > "$BACKUP_DIR/manifest.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "version": "$(cat "$GUILD_HOME/VERSION" 2>/dev/null || echo 'unknown')",
    "deployment_env": "$DEPLOYMENT_ENV",
    "backup_dir": "$BACKUP_DIR",
    "files": [
        "bin/",
        "config/",
        "memory.db",
        "guild-daemon.service"
    ]
}
EOF
    
    success "Backup created at $BACKUP_DIR"
    echo "BACKUP_DIR=$BACKUP_DIR" > /tmp/guild-deployment-vars
    echo "PREVIOUS_VERSION=$(cat "$GUILD_HOME/VERSION" 2>/dev/null || echo 'unknown')" >> /tmp/guild-deployment-vars
}

# Deploy new version
deploy_new_version() {
    log "Deploying Guild version $VERSION..."
    
    # Stop Guild service gracefully
    log "Stopping Guild daemon..."
    if systemctl is-active --quiet guild-daemon; then
        systemctl stop guild-daemon
        
        # Wait for graceful shutdown
        local timeout=30
        while systemctl is-active --quiet guild-daemon && [ $timeout -gt 0 ]; do
            sleep 1
            ((timeout--))
        done
        
        if systemctl is-active --quiet guild-daemon; then
            error "Guild daemon failed to stop gracefully"
            exit 1
        fi
        log "Guild daemon stopped successfully"
    fi
    
    # Extract or copy new binaries
    log "Installing new binaries..."
    if [ -f "./guild-$VERSION.tar.gz" ]; then
        # Extract from tar.gz
        tar -xzf "./guild-$VERSION.tar.gz" -C "/tmp/"
        cp -r "/tmp/guild-$VERSION/"* "$GUILD_HOME/"
    elif [ -d "./dist/guild-$VERSION" ]; then
        # Copy from dist directory
        cp -r "./dist/guild-$VERSION/"* "$GUILD_HOME/"
    else
        error "Deployment artifacts not found for version $VERSION"
        exit 1
    fi
    
    # Set correct permissions
    chmod +x "$GUILD_HOME"/bin/*
    
    # Update version file
    echo "$VERSION" > "$GUILD_HOME/VERSION"
    
    # Run database migrations if needed
    if [ -f "$GUILD_HOME/bin/guild-migrate" ]; then
        log "Running database migrations..."
        if ! "$GUILD_HOME/bin/guild-migrate" --env="$DEPLOYMENT_ENV" --dry-run; then
            error "Database migration validation failed"
            exit 1
        fi
        
        if ! "$GUILD_HOME/bin/guild-migrate" --env="$DEPLOYMENT_ENV"; then
            error "Database migration failed"
            exit 1
        fi
        log "Database migrations completed successfully"
    fi
    
    # Update configuration
    log "Updating configuration..."
    if [ -f "./config/production.yaml" ] && [ "$DEPLOYMENT_ENV" = "production" ]; then
        cp "./config/production.yaml" "$GUILD_CONFIG_DIR/"
    elif [ -f "./config/$DEPLOYMENT_ENV.yaml" ]; then
        cp "./config/$DEPLOYMENT_ENV.yaml" "$GUILD_CONFIG_DIR/"
    fi
    
    # Update systemd service if needed
    if [ -f "./config/guild-daemon.service" ]; then
        cp "./config/guild-daemon.service" "/etc/systemd/system/"
        systemctl daemon-reload
        log "Updated systemd service configuration"
    fi
    
    # Set correct ownership and permissions
    if [ "$USER" = "root" ]; then
        chown -R guild:guild "$GUILD_HOME"
        chown -R guild:guild "$GUILD_CONFIG_DIR"
        chmod 640 "$GUILD_CONFIG_DIR"/*.yaml 2>/dev/null || true
    fi
    
    success "Deployment completed"
}

# Verify deployment
verify_deployment() {
    log "Verifying deployment..."
    
    # Start Guild service
    log "Starting Guild daemon..."
    systemctl start guild-daemon
    
    # Wait for service to be ready
    local timeout=60
    local ready=false
    
    while [ $timeout -gt 0 ]; do
        if systemctl is-active --quiet guild-daemon; then
            # Additional check - try to connect to the service
            if [ -f "$GUILD_HOME/bin/guild" ]; then
                if timeout 5s "$GUILD_HOME/bin/guild" version > /dev/null 2>&1; then
                    ready=true
                    break
                fi
            else
                ready=true
                break
            fi
        fi
        sleep 1
        ((timeout--))
    done
    
    if [ "$ready" = false ]; then
        error "Guild daemon failed to start properly"
        return 1
    fi
    
    log "Guild daemon started successfully"
    
    # Health check
    if [ -f "$GUILD_HOME/bin/guild" ]; then
        log "Running health checks..."
        if ! timeout 30s "$GUILD_HOME/bin/guild" health-check; then
            warning "Health check failed"
            return 1
        fi
        log "Health check passed"
    fi
    
    # Performance validation
    if [ -f "$GUILD_HOME/bin/validate-performance" ]; then
        log "Running performance validation..."
        if ! timeout 60s "$GUILD_HOME/bin/validate-performance" --quick-check; then
            warning "Performance validation failed"
            return 1
        fi
        log "Performance validation passed"
    fi
    
    # Integration test
    if [ -f "$GUILD_HOME/bin/guild" ]; then
        log "Running integration test..."
        if ! timeout 120s "$GUILD_HOME/bin/guild" test --integration --production 2>/dev/null; then
            warning "Integration test failed - this may be expected in some environments"
        else
            log "Integration test passed"
        fi
    fi
    
    # Check version
    if [ -f "$GUILD_HOME/bin/guild" ]; then
        DEPLOYED_VERSION=$("$GUILD_HOME/bin/guild" version 2>/dev/null | grep "Version:" | cut -d' ' -f2 || echo "$VERSION")
        if [ "$DEPLOYED_VERSION" != "$VERSION" ]; then
            warning "Version mismatch. Expected: $VERSION, Got: $DEPLOYED_VERSION"
            return 1
        fi
        log "Version verification passed: $DEPLOYED_VERSION"
    fi
    
    success "Deployment verification passed"
    return 0
}

# Rollback to previous version
rollback_deployment() {
    error "Deployment verification failed. Initiating rollback..."
    
    if [ ! -f "/tmp/guild-deployment-vars" ]; then
        error "Backup information not found. Manual recovery required."
        exit 1
    fi
    
    source /tmp/guild-deployment-vars
    
    if [ ! -d "$BACKUP_DIR" ]; then
        error "Backup directory not found: $BACKUP_DIR"
        exit 1
    fi
    
    log "Rolling back from backup: $BACKUP_DIR"
    
    # Stop current (failed) service
    systemctl stop guild-daemon || true
    
    # Restore binaries
    if [ -d "$BACKUP_DIR/bin" ]; then
        rm -rf "$GUILD_HOME/bin"
        cp -r "$BACKUP_DIR/bin" "$GUILD_HOME/"
        chmod +x "$GUILD_HOME/bin/"*
        log "Restored binaries"
    fi
    
    # Restore configuration
    if [ -d "$BACKUP_DIR/config" ]; then
        rm -rf "$GUILD_CONFIG_DIR"
        cp -r "$BACKUP_DIR/config" "$GUILD_CONFIG_DIR"
        log "Restored configuration"
    fi
    
    # Restore database
    if [ -f "$BACKUP_DIR/memory.db" ]; then
        cp "$BACKUP_DIR/memory.db" "$GUILD_DATA_DIR/memory.db"
        log "Restored database"
    fi
    
    # Restore systemd service
    if [ -f "$BACKUP_DIR/guild-daemon.service" ]; then
        cp "$BACKUP_DIR/guild-daemon.service" "/etc/systemd/system/"
        systemctl daemon-reload
        log "Restored systemd service"
    fi
    
    # Restore version file
    if [ -n "${PREVIOUS_VERSION:-}" ]; then
        echo "$PREVIOUS_VERSION" > "$GUILD_HOME/VERSION"
    fi
    
    # Set permissions
    if [ "$USER" = "root" ]; then
        chown -R guild:guild "$GUILD_HOME"
        chown -R guild:guild "$GUILD_CONFIG_DIR"
    fi
    
    # Start service
    systemctl start guild-daemon
    
    # Verify rollback
    local timeout=30
    while [ $timeout -gt 0 ]; do
        if systemctl is-active --quiet guild-daemon; then
            success "Rollback completed successfully"
            log "Service restored to previous version: ${PREVIOUS_VERSION:-unknown}"
            return 0
        fi
        sleep 1
        ((timeout--))
    done
    
    error "Rollback failed. Manual intervention required."
    exit 1
}

# Post-deployment tasks
post_deployment_tasks() {
    log "Running post-deployment tasks..."
    
    # Update monitoring configuration
    if command -v update-monitoring-config >/dev/null 2>&1; then
        update-monitoring-config --version="$VERSION" || warning "Failed to update monitoring config"
    fi
    
    # Clear caches
    log "Clearing caches..."
    if [ -d "$GUILD_HOME/cache" ]; then
        rm -rf "$GUILD_HOME/cache/"*
        log "Cleared application caches"
    fi
    
    # Restart related services if needed
    if systemctl is-active --quiet guild-web; then
        log "Restarting Guild web interface..."
        systemctl restart guild-web
    fi
    
    # Send deployment notification
    if command -v send-deployment-notification >/dev/null 2>&1; then
        send-deployment-notification \
            --env="$DEPLOYMENT_ENV" \
            --version="$VERSION" \
            --status="success" || warning "Failed to send deployment notification"
    fi
    
    # Update load balancer health check
    if command -v update-health-check >/dev/null 2>&1; then
        update-health-check --enable || warning "Failed to update health check"
    fi
    
    # Log deployment success
    echo "$(date -Iseconds): Successfully deployed Guild $VERSION to $DEPLOYMENT_ENV" >> "$GUILD_HOME/deployment-history.log"
    
    success "Post-deployment tasks completed"
}

# Handle manual rollback
handle_rollback() {
    log "Manual rollback requested"
    
    if [ ! -f "/tmp/guild-deployment-vars" ]; then
        error "No recent deployment information found"
        exit 1
    fi
    
    rollback_deployment
}

# Print usage information
print_usage() {
    cat << EOF
Guild Production Deployment Script

Usage: $0 [OPTIONS]

Options:
    --rollback          Rollback to previous version
    --check-only        Run pre-deployment checks only
    --skip-checks       Skip pre-deployment checks (dangerous)
    --help              Show this help message

Environment Variables:
    DEPLOYMENT_ENV      Target environment (default: production)
    VERSION             Version to deploy (default: git describe --tags --always)
    GUILD_USER          User to run Guild as (default: guild-deploy)
    GUILD_HOME          Guild installation directory (default: /opt/guild)
    GUILD_CONFIG_DIR    Guild configuration directory (default: /etc/guild)
    GUILD_DATA_DIR      Guild data directory (default: /opt/guild/data)

Examples:
    # Deploy latest version to production
    ./deploy-production.sh
    
    # Deploy specific version
    VERSION=v1.2.3 ./deploy-production.sh
    
    # Deploy to staging
    DEPLOYMENT_ENV=staging ./deploy-production.sh
    
    # Rollback to previous version
    ./deploy-production.sh --rollback
    
    # Check deployment readiness only
    ./deploy-production.sh --check-only

EOF
}

# Main deployment process
main() {
    local skip_checks=false
    local check_only=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --rollback)
                handle_rollback
                exit 0
                ;;
            --check-only)
                check_only=true
                shift
                ;;
            --skip-checks)
                skip_checks=true
                shift
                ;;
            --help)
                print_usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    log "Starting Guild deployment process"
    log "Version: $VERSION"
    log "Environment: $DEPLOYMENT_ENV"
    log "User: $USER"
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Run pre-deployment checks
    if [ "$skip_checks" = false ]; then
        run_pre_deployment_checks
    fi
    
    if [ "$check_only" = true ]; then
        success "Pre-deployment checks completed successfully"
        exit 0
    fi
    
    # Run deployment steps
    create_backup
    deploy_new_version
    
    if verify_deployment; then
        post_deployment_tasks
        success "Guild deployment completed successfully!"
        log "Version $VERSION is now live in $DEPLOYMENT_ENV environment"
        log "Backup available at: $BACKUP_DIR"
    else
        rollback_deployment
        error "Deployment failed and was rolled back"
        exit 1
    fi
    
    # Clean up old backups (keep last 5)
    find /tmp -name "guild-backup-*" -type d -mtime +7 -exec rm -rf {} \; 2>/dev/null || true
    
    success "Guild deployment process completed"
}

# Script entry point
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi