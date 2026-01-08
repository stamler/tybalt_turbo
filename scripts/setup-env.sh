#!/bin/bash

# Script to set up environment variables for litestream operations
# Usage: source scripts/setup-env.sh

# Check if script is being sourced (required for setting env vars)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "❌ This script must be sourced, not executed directly."
    echo "💡 Use: source scripts/setup-env.sh"
    echo "💡 Or from project root: source scripts/setup-env.sh"
    exit 1
fi

echo "🔧 Setting up litestream environment variables..."

# Find the project root directory
if [[ "${BASH_SOURCE[0]}" == *"scripts/setup-env.sh" ]]; then
    # Script is being sourced with full path or from scripts dir
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
else
    # Try to find project root by looking for fly.toml
    CURRENT_DIR="$(pwd)"
    while [[ "$CURRENT_DIR" != "/" ]]; do
        if [[ -f "$CURRENT_DIR/fly.toml" ]]; then
            PROJECT_ROOT="$CURRENT_DIR"
            break
        fi
        CURRENT_DIR="$(dirname "$CURRENT_DIR")"
    done
    
    if [[ -z "$PROJECT_ROOT" ]]; then
        echo "❌ Could not find project root (looking for fly.toml)"
        return 1
    fi
fi

# Change to project root where fly.toml should be
cd "$PROJECT_ROOT"

echo "📁 Working from: $PROJECT_ROOT"

# Check if fly.toml exists
if [ ! -f "fly.toml" ]; then
    echo "❌ fly.toml not found. Are you in the correct project directory?"
    return 1
fi

# Get app info
echo "🔑 Fetching secrets from Fly.io..."
APP_NAME=$(grep '^app = ' fly.toml | sed 's/app = "\(.*\)"/\1/' 2>/dev/null)
if [ -z "$APP_NAME" ]; then
    echo "❌ Could not determine app name from fly.toml."
    return 1
fi

echo "📱 App: $APP_NAME"

# Get a machine ID to exec commands on
MACHINE_ID=$(flyctl status --json 2>/dev/null | jq -r '.Machines[0].id' 2>/dev/null)

if [ "$MACHINE_ID" = "null" ] || [ -z "$MACHINE_ID" ]; then
    echo "❌ No machines found. Make sure your app is deployed and running."
    echo "💡 Try: flyctl machine start"
    return 1
fi

echo "🖥️  Using machine: $MACHINE_ID"

# Function to get secret value
get_secret() {
    local secret_name=$1
    flyctl machine exec $MACHINE_ID "printenv $secret_name" 2>/dev/null || echo ""
}

# Set environment variables
export LITESTREAM_ACCESS_KEY_ID=$(get_secret LITESTREAM_ACCESS_KEY_ID)
export LITESTREAM_SECRET_ACCESS_KEY=$(get_secret LITESTREAM_SECRET_ACCESS_KEY)
export LITESTREAM_BUCKET=$(get_secret LITESTREAM_BUCKET)
export LITESTREAM_REGION=$(get_secret LITESTREAM_REGION)
export LITESTREAM_ENDPOINT=$(get_secret LITESTREAM_ENDPOINT)

# Validate we got the secrets
if [ -z "$LITESTREAM_BUCKET" ]; then
    echo "❌ Failed to fetch secrets. Make sure your app is running and you're authenticated with flyctl."
    return 1
fi

echo "✅ Environment variables set:"
echo "   LITESTREAM_BUCKET: $LITESTREAM_BUCKET"
echo "   LITESTREAM_REGION: ${LITESTREAM_REGION:-us-east-1}"
echo "   LITESTREAM_ENDPOINT: ${LITESTREAM_ENDPOINT:-https://fly.storage.tigris.dev}"
echo ""
echo "🚀 You can now run litestream commands:"
echo "   litestream snapshots -config litestream.local.yml app/pb_data/data.db"
echo "   litestream restore -config litestream.local.yml -o ~/prod-backup.db app/pb_data/data.db" 