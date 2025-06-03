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
echo "📊 Starting replication with real-time monitoring..."
echo "📝 Watch the logs below - when you see replication activity slow down, press Ctrl-C to cancel replication and restart the app"
echo "----------------------------------------"

# Set up signal trapping for graceful Ctrl-C handling
cleanup_and_continue() {
    echo ""
    echo "----------------------------------------"
    echo "🛑 Stopping replication..."
    
    # Kill the litestream process
    if [ ! -z "$REPLICATE_PID" ] && kill -0 $REPLICATE_PID 2>/dev/null; then
        kill $REPLICATE_PID 2>/dev/null
        wait $REPLICATE_PID 2>/dev/null
    fi
    
    echo "✅ Replication stopped"
    echo "🧹 Cleaning up..."
    
    # Remove temp log if it exists
    [ -f "$TEMP_LOG" ] && rm -f "$TEMP_LOG"
    
    echo "✅ Cleanup complete"
    echo ""
    echo "📋 Next step: Starting production app..."
    
    # Continue with the rest of the script
    start_production_app
}

start_production_app() {
    echo "🚀 Starting production app..."
    flyctl machine start $MACHINE_ID
    echo "✅ Production app started!"

    echo ""
    echo "🎉 Deployment complete!"
    echo "🌐 Check your app at: https://$(grep '^app = ' fly.toml | sed 's/app = "\(.*\)"/\1/').fly.dev"
}

# Set up the signal trap
trap cleanup_and_continue SIGINT

# Use a temporary log file to capture litestream output and show it in real-time
TEMP_LOG="/tmp/litestream_deploy_$$.log"

# Start litestream and show output in real-time
litestream replicate -config "$LOCAL_CONFIG" 2>&1 | tee "$TEMP_LOG" &
REPLICATE_PID=$!

# Wait for the process (or Ctrl-C)
set +e  # Temporarily disable exit on error so SIGINT doesn't cause immediate exit
wait $REPLICATE_PID 2>/dev/null
WAIT_EXIT_CODE=$?
set -e  # Re-enable exit on error

# If we get here without Ctrl-C, the process exited on its own (shouldn't happen)
if [ $WAIT_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "⚠️  Replication process exited unexpectedly"
    echo "📋 Log contents:"
    cat "$TEMP_LOG" 2>/dev/null
    cleanup_and_continue
fi

# If we get here, it means we were interrupted by SIGINT
# The trap will handle cleanup automatically 