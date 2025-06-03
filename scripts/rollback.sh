#!/bin/bash
set -e

# Find the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Usage: ./scripts/rollback.sh [generation_id]
GENERATION=${1:-}

if [ -z "$GENERATION" ]; then
    echo "Usage: $0 <generation_id>"
    echo ""
    echo "This script combines restore + deploy for quick rollbacks."
    echo "For more control, use the individual scripts:"
    echo "   • ./scripts/restore-generation.sh <generation_id>"
    echo "   • ./scripts/deploy-local-db.sh"
    echo ""
    echo "Available generations:"
    "$SCRIPT_DIR/list-generations.sh"
    exit 1
fi

echo "🔄 Rolling back to generation: $GENERATION"
echo "   Step 1: Restoring generation locally..."
echo "   Step 2: Deploying to production..."
echo ""

# Ensure we're working from the project root
cd "$PROJECT_ROOT"

# Step 1: Restore generation locally
"$SCRIPT_DIR/restore-generation.sh" "$GENERATION"

echo ""
echo "⏳ Proceeding to deployment in 5 seconds... (Ctrl+C to cancel)"
sleep 5

# Step 2: Deploy to production (with confirmation)
"$SCRIPT_DIR/deploy-local-db.sh"

echo ""
echo "✅ Rollback complete!" 