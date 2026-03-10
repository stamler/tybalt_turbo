#!/bin/bash
set -e
set -o pipefail

###############################################################################
# Deploy a local SQLite database to production via the Litestream replica.
#
# This script exists for the "replace production DB" workflow, not normal app
# deploys. It is intended for cases such as:
# - pushing a locally-prepared database to production
# - rolling production back to a known-good local copy
# - replacing a corrupted production database
#
# Current production topology assumptions:
# - Fly Machines deployment
# - exactly one running writable machine
# - production SQLite database mounted at /app/pb_data/data.db on a Fly volume
# - Litestream replicating that database to S3-compatible object storage
#
# Why this needs a special workflow:
# - Normal restarts now reuse the on-volume database.
# - That means simply syncing a new DB to S3 and restarting is not enough,
#   because startup would otherwise keep using the existing local DB.
# - The production startup script supports an explicit override via
#   /app/pb_data/.force-restore. When present, startup:
#   - deletes the on-volume DB
#   - deletes WAL/SHM files
#   - deletes local Litestream state
#   - restores a fresh DB from the Litestream replica in S3
#
# What this script does:
# 1. Validates Litestream credentials are available locally.
# 2. Confirms a local source database exists.
# 3. Confirms there is exactly one running Fly machine.
# 4. Marks production for forced restore on next boot.
# 5. Stops the running production machine and waits for it to actually stop.
# 6. Verifies there are zero running machines before proceeding.
# 7. Starts local `litestream replicate` so the replacement DB is pushed to S3.
# 8. Streams Litestream output so the operator can watch replication progress.
# 9. On Ctrl-C, stops local replication and restarts production.
#
# Important safety properties:
# - Production is stopped before the replacement DB is pushed, so the live app
#   cannot continue writing to the old DB while the replica contents change.
# - The script refuses to continue if machine state is ambiguous.
# - If local replication exits unexpectedly or fails, production is NOT started
#   automatically. This avoids booting with `.force-restore` still armed and
#   restoring stale or incomplete replica contents into the mounted volume.
#
# Operator model:
# - Run the script.
# - Watch Litestream output.
# - Once the replica looks fully caught up, press Ctrl-C.
# - The SIGINT trap treats Ctrl-C as the intentional "cut over now" signal,
#   stops local replication, and restarts production.
###############################################################################

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
    echo "💡 Tip: Restore from backup first with: litestream restore -config litestream.local.yml app/pb_data/data.db"
    exit 1
fi

# Show database info
echo "📁 Local database: $LOCAL_DB"
echo "📊 Database size: $(du -h "$LOCAL_DB" | cut -f1)"
echo ""

# This operation is intentionally destructive from production's point of view,
# so require an explicit interactive confirmation before touching the machine.
read -p "⚠️  This will replace the ENTIRE production database. Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Deployment cancelled."
    exit 1
fi

# Query Fly once up front and verify the deployment is in the single-machine
# shape this workflow expects. If there are zero or multiple running machines,
# the safe thing is to stop and ask the operator to resolve that state first.
STATUS_JSON=$(flyctl status --json 2>/dev/null) || {
    echo "❌ Could not query Fly machine status"
    exit 1
}

RUNNING_MACHINE_COUNT=$(printf '%s\n' "$STATUS_JSON" | jq '[.Machines[] | select(((.state // .State // "") | ascii_downcase) == "started")] | length' 2>/dev/null)
if [ -z "$RUNNING_MACHINE_COUNT" ]; then
    echo "❌ Could not determine running machine count"
    exit 1
fi

if [ "$RUNNING_MACHINE_COUNT" -ne 1 ]; then
    echo "❌ Expected exactly one running machine, found $RUNNING_MACHINE_COUNT"
    echo "💡 This script is only safe for a single-machine deployment."
    echo "   Resolve the extra machine state manually, then rerun."
    exit 1
fi

MACHINE_ID=$(printf '%s\n' "$STATUS_JSON" | jq -r '.Machines[] | select(((.state // .State // "") | ascii_downcase) == "started") | .id' 2>/dev/null)
if [ -z "$MACHINE_ID" ] || [ "$MACHINE_ID" = "null" ]; then
    echo "❌ Could not determine running machine ID"
    exit 1
fi

# Re-query Fly after a stop request and count machines still in the "started"
# state. We use this as a postcondition check before changing replica contents.
count_running_machines() {
    local status_json
    status_json=$(flyctl status --json 2>/dev/null) || return 1
    printf '%s\n' "$status_json" | jq '[.Machines[] | select(((.state // .State // "") | ascii_downcase) == "started")] | length'
}

# Mark production so the next startup performs a clean restore from S3 instead
# of reusing the existing on-volume database. We set this BEFORE stopping the
# machine so the flag is present when the machine later comes back up.
echo "📝 Marking production to restore DB on next boot..."
flyctl ssh console -C "mkdir -p /app/pb_data && touch /app/pb_data/.force-restore" >/dev/null

# Stop production before changing the authoritative replica contents. The
# built-in wait avoids returning immediately while the machine is still in the
# process of shutting down.
echo "⏹️  Stopping production app..."
flyctl machine stop "$MACHINE_ID" --wait-timeout 2m

# Confirm the stop actually took effect. If any machine is still running, bail
# out rather than risk writing a replacement replica while production is live.
RUNNING_AFTER_STOP=$(count_running_machines) || {
    echo "❌ Could not verify machine state after stop"
    exit 1
}

if [ "$RUNNING_AFTER_STOP" -ne 0 ]; then
    echo "❌ Expected zero running machines after stop, found $RUNNING_AFTER_STOP"
    echo "💡 Refusing to continue while production may still be writing to the database."
    exit 1
fi

# At this point production is down and the next boot is armed for a clean
# restore. The remaining task is to push the local DB into the replica path
# that startup will restore from.
echo "☁️  Pushing local database to S3..."
echo "📊 Starting replication with real-time monitoring..."
echo "📝 Watch the logs below - when you see replication activity slow down, press Ctrl-C to cancel replication and restart the app"
echo "----------------------------------------"

# SIGINT is the operator's "cut over now" signal. When they press Ctrl-C after
# replication appears caught up, stop local Litestream cleanly and restart the
# production machine so it restores from the new replica contents.
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
    
    # Once local replication has been interrupted intentionally, production can
    # boot and restore the replacement DB from S3 into the mounted volume.
    start_production_app
    exit 0
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

# Capture Litestream output to both the terminal and a temp file so we can show
# the operator what happened if the process exits unexpectedly.
TEMP_LOG="/tmp/litestream_deploy_$$.log"

# Start litestream and show output in real-time. Process substitution ensures
# REPLICATE_PID is the litestream process, not the tee helper.
litestream replicate -config "$LOCAL_CONFIG" > >(tee "$TEMP_LOG") 2>&1 &
REPLICATE_PID=$!

# Wait for Litestream until one of two things happens:
# - the operator presses Ctrl-C, in which case the SIGINT trap handles cutover
# - Litestream exits on its own, which is treated as a failure/suspicious state
set +e  # Temporarily disable exit on error so SIGINT doesn't cause immediate exit
wait $REPLICATE_PID 2>/dev/null
WAIT_EXIT_CODE=$?
set -e  # Re-enable exit on error

# A clean Litestream exit is not expected here because replication normally
# runs continuously. Do not restart production automatically in this case;
# require manual inspection first so we do not restore stale data by accident.
if [ $WAIT_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "⚠️  Replication process exited unexpectedly"
    echo "📋 Log contents:"
    cat "$TEMP_LOG" 2>/dev/null
    echo "❌ Not restarting production automatically."
    echo "💡 Verify the replica contents before starting production."
    exit 1
fi

# Any non-zero exit means replication failed. Leave production stopped and keep
# the failure loud so the operator can either retry or deliberately clear the
# force-restore flag before bringing the old production DB back.
echo ""
echo "❌ Replication failed with exit code $WAIT_EXIT_CODE"
echo "📋 Log contents:"
cat "$TEMP_LOG" 2>/dev/null
echo "❌ Production remains stopped so the existing mounted database is not overwritten by a bad restore."
echo "💡 If you need to bring production back on the old database, clear /app/pb_data/.force-restore before starting the machine."
exit 1
