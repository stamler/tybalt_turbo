#!/bin/bash
set -e

# Find the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOCAL_CONFIG="$PROJECT_ROOT/litestream.local.yml"
LOCAL_DB="$PROJECT_ROOT/app/pb_data/data.db"

echo "🚀 Deploying local database to production..."

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

# Check if local database exists
if [ ! -f "$LOCAL_DB" ]; then
    echo "❌ No local database found at: $LOCAL_DB"
    echo "💡 Tip: Restore a generation first with: ./scripts/restore-generation.sh <generation_id>"
    exit 1
fi

# Show database info
echo "📁 Local database: $LOCAL_DB"
echo "📊 Database size: $(du -h "$LOCAL_DB" | cut -f1)"
echo ""

# Confirm deployment
read -p "⚠️  This will replace the ENTIRE production database. Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Deployment cancelled."
    exit 1
fi

# Get machine ID first
MACHINE_ID=$(flyctl status --json 2>/dev/null | jq -r '.Machines[0].id' 2>/dev/null)
if [ -z "$MACHINE_ID" ] || [ "$MACHINE_ID" = "null" ]; then
    echo "❌ Could not determine machine ID"
    echo "💡 Please stop the machine manually before deploying:"
    echo "   flyctl machine stop"
    exit 1
fi

# Stop production app to prevent race conditions with S3 replication
echo "⏹️  Stopping production app..."
flyctl machine stop $MACHINE_ID

# Push to production
echo "☁️  Pushing local database to S3..."
litestream replicate -config "$LOCAL_CONFIG" &
REPLICATE_PID=$!

# Wait for replication to start (give it 20 seconds, but this is a guess)
sleep 20

# Kill the replication process (it would run forever otherwise)
kill $REPLICATE_PID 2>/dev/null || true

echo "✅ Database pushed to S3"

# Start production app - it will restore from our new backup
echo "🚀 Starting production app..."
flyctl machine start $MACHINE_ID
echo "✅ Production app started!"

echo ""
echo "🎉 Deployment complete!"
echo "🌐 Check your app at: https://$(grep '^app = ' fly.toml | sed 's/app = "\(.*\)"/\1/').fly.dev" 