#!/bin/bash
set -e

# Find the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOCAL_CONFIG="$PROJECT_ROOT/litestream.local.yml"
LOCAL_DB="$PROJECT_ROOT/app/pb_data/data.db"

# Usage: ./scripts/rollback.sh [generation_id]
GENERATION=${1:-}

if [ -z "$GENERATION" ]; then
    echo "Usage: $0 <generation_id>"
    echo ""
    echo "Available generations:"
    if [ -f "$LOCAL_CONFIG" ]; then
        litestream generations -config "$LOCAL_CONFIG" 2>/dev/null || litestream generations -replica s3://${LITESTREAM_BUCKET}/tybalt
    else
        litestream generations -replica s3://${LITESTREAM_BUCKET}/tybalt 2>/dev/null || echo "❌ No config found. Run 'source scripts/setup-env.sh' first"
    fi
    exit 1
fi

echo "🔄 Rolling back to generation: $GENERATION"

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
echo "📥 Restoring generation $GENERATION locally..."
litestream restore -config "$LOCAL_CONFIG" -generation $GENERATION "$LOCAL_DB"

# Push to production
echo "☁️  Pushing restored database to S3..."
litestream replicate -config "$LOCAL_CONFIG" &
REPLICATE_PID=$!

# Wait for replication to start (give it 10 seconds)
sleep 10

# Kill the replication process (it would run forever otherwise)
kill $REPLICATE_PID 2>/dev/null || true

echo "✅ Database pushed to S3"

# Restart production app
echo "🚀 Restarting production app..."
MACHINE_ID=$(flyctl status --json 2>/dev/null | jq -r '.Machines[0].id' 2>/dev/null)
if [ -n "$MACHINE_ID" ] && [ "$MACHINE_ID" != "null" ]; then
    flyctl machine restart $MACHINE_ID
else
    echo "⚠️  Could not determine machine ID, please restart manually:"
    echo "   flyctl machine restart"
fi

echo "✅ Rollback complete!"
echo "🌐 Check your app at: https://$(grep '^app = ' fly.toml | sed 's/app = "\(.*\)"/\1/').fly.dev" 