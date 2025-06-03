#!/bin/bash

# Script to list available database generations
echo "🗂️  Available database generations:"
echo ""

# Find the project root (where litestream.local.yml should be)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOCAL_CONFIG="$PROJECT_ROOT/litestream.local.yml"
LOCAL_DB="$PROJECT_ROOT/app/pb_data/data.db"

# Use local config to list generations
if [ -f "$LOCAL_CONFIG" ] && litestream generations -config "$LOCAL_CONFIG" "$LOCAL_DB" 2>/dev/null; then
    echo ""
    echo "💡 To rollback: ./scripts/rollback.sh <generation_id>"
else
    echo "❌ No litestream access configured."
    echo "💡 Run: source scripts/setup-env.sh"
fi 