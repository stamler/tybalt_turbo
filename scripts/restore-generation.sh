#!/bin/bash
set -e

# Find the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOCAL_CONFIG="$PROJECT_ROOT/litestream.local.yml"
LOCAL_DB="$PROJECT_ROOT/app/pb_data/data.db"

# Usage: ./scripts/restore-generation.sh [generation_id]
GENERATION=${1:-}

if [ -z "$GENERATION" ]; then
    echo "Usage: $0 <generation_id>"
    echo ""
    echo "Available generations:"
    if [ -f "$LOCAL_CONFIG" ]; then
        litestream generations -config "$LOCAL_CONFIG" "$LOCAL_DB" 2>/dev/null || echo "❌ No litestream access. Run 'source scripts/setup-env.sh' first"
    else
        echo "❌ No config found. Run 'source scripts/setup-env.sh' first"
    fi
    exit 1
fi

echo "📥 Restoring generation $GENERATION to local database..."

# Validate environment variables
required_vars=("LITESTREAM_ACCESS_KEY_ID" "LITESTREAM_SECRET_ACCESS_KEY" "LITESTREAM_BUCKET")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "❌ Error: $var environment variable is not set"
        echo "💡 Tip: Run 'source scripts/setup-env.sh' first"
        exit 1
    fi
done

# Ensure we're working from the project root
cd "$PROJECT_ROOT"

# Backup current local database
if [ -f "$LOCAL_DB" ]; then
    echo "💾 Backing up current local database..."
    cp "$LOCAL_DB" "$LOCAL_DB.backup.$(date +%s)"
    # Remove the original file so litestream can restore to it
    rm "$LOCAL_DB"
fi

# Restore specific generation locally
echo "⬇️  Downloading generation $GENERATION..."
litestream restore -config "$LOCAL_CONFIG" -generation $GENERATION "$LOCAL_DB"

echo "✅ Generation $GENERATION restored to local database!"
echo "📁 Database location: $LOCAL_DB"
echo ""
echo "🔄 Next steps:"
echo "   • Test locally: cd app && go run main.go serve"
echo "   • Deploy to production: ./scripts/deploy-local-db.sh"
echo "   • List generations: ./scripts/list-generations.sh" 