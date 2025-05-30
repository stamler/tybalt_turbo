#!/bin/bash

# Tybalt Production Deployment Script
# This script handles database backups, migrations, and safe deployments

set -e  # Exit on any error

# Configuration
BACKUP_DIR="./backups"
DATA_DIR="./data"
COMPOSE_FILE="docker-compose.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

create_backup() {
    local backup_name="backup-$(date +%Y%m%d-%H%M%S)"
    local backup_path="$BACKUP_DIR/$backup_name.tar.gz"
    
    log_info "Creating backup: $backup_name"
    
    # Create backup directory if it doesn't exist
    mkdir -p "$BACKUP_DIR"
    
    # Stop application for consistent backup
    log_info "Stopping application for backup..."
    docker-compose down
    
    # Create backup
    tar -czf "$backup_path" "$DATA_DIR"
    
    log_info "Backup created: $backup_path"
    echo "$backup_path"  # Return backup path
}

check_migrations() {
    log_info "Checking for pending migrations..."
    
    # Build new image to check migrations
    docker-compose build --quiet
    
    # Check migration status
    local pending=$(docker-compose run --rm tybalt ./tybalt migrate list 2>/dev/null | grep -c "pending" || echo "0")
    
    if [ "$pending" -gt 0 ]; then
        log_warn "Found $pending pending migration(s)"
        return 0
    else
        log_info "No pending migrations"
        return 1
    fi
}

deploy() {
    local backup_path=""
    local rollback_available=false
    
    # Check if we're in the right directory
    if [ ! -f "$COMPOSE_FILE" ]; then
        log_error "docker-compose.yml not found. Are you in the right directory?"
        exit 1
    fi
    
    # Check if data directory exists
    if [ ! -d "$DATA_DIR" ]; then
        log_error "Data directory not found: $DATA_DIR"
        exit 1
    fi
    
    log_info "Starting deployment process..."
    
    # Create backup
    backup_path=$(create_backup)
    rollback_available=true
    
    # Build new image
    log_info "Building new application image..."
    docker-compose build
    
    # Check for pending migrations
    if check_migrations; then
        log_warn "This deployment includes database migrations"
        echo
        read -p "Continue with deployment? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Deployment cancelled by user"
            # Restart previous version
            docker-compose up -d
            exit 0
        fi
    fi
    
    # Start new version
    log_info "Starting new application version..."
    docker-compose up -d
    
    # Wait for application to be ready
    log_info "Waiting for application to be ready..."
    sleep 10
    
    # Check if application is healthy
    if ! docker-compose ps | grep -q "Up"; then
        log_error "Application failed to start!"
        
        if [ "$rollback_available" = true ]; then
            log_warn "Attempting automatic rollback..."
            rollback "$backup_path"
        fi
        exit 1
    fi
    
    # Verify application health
    log_info "Verifying application health..."
    
    # Try to connect to the application
    local max_attempts=12
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s http://localhost:8080/api/health >/dev/null 2>&1; then
            log_info "Application is healthy!"
            break
        fi
        
        log_info "Health check attempt $attempt/$max_attempts..."
        sleep 5
        ((attempt++))
    done
    
    if [ $attempt -gt $max_attempts ]; then
        log_error "Application health check failed!"
        
        if [ "$rollback_available" = true ]; then
            log_warn "Attempting automatic rollback..."
            rollback "$backup_path"
        fi
        exit 1
    fi
    
    log_info "Deployment completed successfully!"
    log_info "Backup available at: $backup_path"
    
    # Show logs
    echo
    log_info "Recent application logs:"
    docker-compose logs --tail=20 tybalt
}

rollback() {
    local backup_path="$1"
    
    if [ -z "$backup_path" ]; then
        log_error "No backup path provided for rollback"
        echo
        echo "Available backups:"
        ls -la "$BACKUP_DIR"/*.tar.gz 2>/dev/null || echo "No backups found"
        exit 1
    fi
    
    if [ ! -f "$backup_path" ]; then
        log_error "Backup file not found: $backup_path"
        exit 1
    fi
    
    log_warn "Rolling back to backup: $backup_path"
    
    # Stop current application
    docker-compose down
    
    # Restore backup
    log_info "Restoring data from backup..."
    rm -rf "$DATA_DIR"
    tar -xzf "$backup_path"
    
    # Start previous version
    log_info "Starting previous application version..."
    docker-compose up -d
    
    log_info "Rollback completed"
}

show_status() {
    log_info "Application Status:"
    docker-compose ps
    
    echo
    log_info "Recent Logs:"
    docker-compose logs --tail=10 tybalt
    
    echo
    log_info "Migration Status:"
    docker-compose exec tybalt ./tybalt migrate list || log_warn "Could not check migration status"
    
    echo
    log_info "Available Backups:"
    ls -la "$BACKUP_DIR"/*.tar.gz 2>/dev/null || echo "No backups found"
}

show_help() {
    echo "Tybalt Production Deployment Script"
    echo
    echo "Usage: $0 [command]"
    echo
    echo "Commands:"
    echo "  deploy          Full deployment with backup and migration"
    echo "  backup          Create backup only"
    echo "  rollback PATH   Rollback to specific backup"
    echo "  status          Show application and migration status"
    echo "  help            Show this help message"
    echo
    echo "Examples:"
    echo "  $0 deploy                                    # Deploy latest code"
    echo "  $0 backup                                    # Create backup only"
    echo "  $0 rollback ./backups/backup-20240115.tar.gz # Rollback to specific backup"
    echo "  $0 status                                    # Check status"
}

# Main script logic
case "${1:-deploy}" in
    "deploy")
        deploy
        ;;
    "backup")
        create_backup
        docker-compose up -d  # Restart after backup
        ;;
    "rollback")
        rollback "$2"
        ;;
    "status")
        show_status
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        echo
        show_help
        exit 1
        ;;
esac 