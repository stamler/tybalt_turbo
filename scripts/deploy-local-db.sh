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
#   `LITESTREAM_FORCE_RESTORE=1`. When set on boot, startup:
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
# 5. Temporarily disables Fly auto-start on that machine, then stops it.
# 6. Verifies there are zero running machines before proceeding.
# 7. Clears local Litestream state so a fresh generation is pushed.
# 8. Runs `litestream replicate -once -force-snapshot` to push a complete
#    replacement snapshot to S3 and exit only when that upload is done.
# 9. Re-enables auto-start and starts the same production machine again.
#
# Important safety properties:
# - Production is stopped before the replacement DB is pushed, so the live app
#   cannot continue writing to the old DB while the replica contents change.
# - The script disables machine auto-start before stopping so Fly does not
#   immediately wake the machine back up while the replica contents are in flux.
# - The script stages `LITESTREAM_FORCE_RESTORE=1` for the next boot as the
#   single authoritative restore trigger.
# - The script clears local Litestream state before replication so a replaced
#   local database does not reuse stale generation metadata.
# - The script refuses to continue if machine state is ambiguous.
# - If the one-shot snapshot upload fails, production is NOT started
#   automatically. This avoids booting with restore still armed and pulling
#   stale or incomplete replica contents into the mounted volume.
#
# Operator model:
# - Run the script.
# - Wait for the one-shot Litestream snapshot upload to finish.
# - If that succeeds, the script starts production automatically.
###############################################################################

# Find the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOCAL_CONFIG="$PROJECT_ROOT/litestream.local.yml"
LOCAL_DB="$PROJECT_ROOT/app/pb_data/data.db"
LOCAL_STATE_DIR="$PROJECT_ROOT/app/pb_data/.data.db-litestream"
APP_NAME=$(sed -n 's/^app = "\(.*\)"/\1/p' "$PROJECT_ROOT/fly.toml" | head -n1)
FORCE_RESTORE_SECRET_NAME="LITESTREAM_FORCE_RESTORE"

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

if [ -z "$APP_NAME" ]; then
    echo "❌ Could not determine Fly app name from fly.toml"
    exit 1
fi

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

echo "WARNING: After production starts again, verify the startup logs before trusting the cutover."
echo "WARNING: Good sign: a log line beginning with '[start] Restore requested:'"
echo "WARNING: Bad sign: a log line beginning with '[start] Using existing database at'"
echo "WARNING: If you see the bad sign, the replacement DB was not restored from the replica."
echo ""

# Query Fly once up front and verify the deployment is in the single-machine
# shape this workflow expects. If there are zero or multiple running machines,
# the safe thing is to stop and ask the operator to resolve that state first.
STATUS_JSON=$(flyctl status --json -a "$APP_NAME" 2>/dev/null) || {
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

# Re-query Fly after stopping and count machines still in the "started"
# state. We use this as a postcondition check before changing replica contents.
count_running_machines() {
    local status_json
    status_json=$(flyctl status --json -a "$APP_NAME" 2>/dev/null) || return 1
    printf '%s\n' "$status_json" | jq '[.Machines[] | select(((.state // .State // "") | ascii_downcase) == "started")] | length'
}

machine_state_summary() {
    local status_json
    status_json=$(flyctl status --json -a "$APP_NAME" 2>/dev/null) || return 1
    printf '%s\n' "$status_json" | jq -r '[.Machines[] | "\(.id):\((.state // .State // "unknown") | ascii_downcase)"] | join(", ")'
}

wait_for_zero_running_machines() {
    local max_attempts=24
    local sleep_seconds=5
    local attempt=1
    local running_count
    local state_summary

    while [ "$attempt" -le "$max_attempts" ]; do
        running_count=$(count_running_machines) || {
            echo "❌ Could not verify machine state after stopping production"
            return 1
        }

        if [ "$running_count" -eq 0 ]; then
            state_summary=$(machine_state_summary 2>/dev/null || echo "unknown")
            echo "✅ Fly reports zero running machines"
            echo "📋 Machine states: $state_summary"
            return 0
        fi

        state_summary=$(machine_state_summary 2>/dev/null || echo "unknown")
        echo "⏳ Waiting for production to stop completely... ($attempt/$max_attempts)"
        echo "📋 Machine states: $state_summary"
        sleep "$sleep_seconds"
        attempt=$((attempt + 1))
    done

    running_count=$(count_running_machines 2>/dev/null || echo "unknown")
    state_summary=$(machine_state_summary 2>/dev/null || echo "unknown")
    echo "❌ Expected zero running machines after stopping production, found $running_count"
    echo "📋 Final machine states: $state_summary"
    echo "💡 Refusing to continue while production may still be writing to the database."
    return 1
}

set_machine_autostart() {
    local autostart_value="$1"
    flyctl machine update "$MACHINE_ID" -a "$APP_NAME" --autostart="$autostart_value" --skip-start -y >/dev/null
}

stage_force_restore_secret() {
    flyctl secrets set -a "$APP_NAME" --stage "${FORCE_RESTORE_SECRET_NAME}=1" >/dev/null
}

clear_staged_force_restore_secret() {
    flyctl secrets unset -a "$APP_NAME" --stage "$FORCE_RESTORE_SECRET_NAME" >/dev/null
}

# Mark production so the next startup performs a clean restore from S3 instead
# of reusing the existing on-volume database.
echo "📝 Staging LITESTREAM_FORCE_RESTORE=1 for the next boot..."
stage_force_restore_secret

# Disable auto-start before stopping. Without this, Fly can wake the machine
# back up on inbound traffic before the replacement replica is fully pushed.
echo "🛑 Disabling Fly auto-start on the production machine..."
set_machine_autostart false

# Stop production before changing the authoritative replica contents.
echo "⏹️  Stopping production app..."
flyctl machine stop "$MACHINE_ID" -a "$APP_NAME" --wait-timeout 2m

# Confirm the stop actually took effect. Poll rather than sampling once,
# because Fly can take a short time to transition a machine out of "started".
if ! wait_for_zero_running_machines; then
    exit 1
fi

# At this point production is down and the next boot is armed for a clean
# restore. Reset local Litestream metadata before replicating so a database
# restored or swapped locally is treated as a fresh generation.
echo "🧹 Resetting local Litestream state..."
rm -rf "$LOCAL_STATE_DIR"

echo "☁️  Pushing local database to S3..."
echo "📸 Forcing a complete Litestream snapshot upload..."
echo "----------------------------------------"

start_production_app() {
    echo "🚀 Re-enabling Fly auto-start on the production machine..."
    set_machine_autostart true
    echo "🚀 Starting production app..."
    flyctl machine start "$MACHINE_ID" -a "$APP_NAME"
    echo "🧹 Clearing staged restore secret for future boots..."
    clear_staged_force_restore_secret
    echo "✅ Production app started!"

    echo ""
    echo "🎉 Deployment complete!"
    echo "🌐 Check your app at: https://$APP_NAME.fly.dev"
}

# Capture Litestream output to both the terminal and a temp file so we can show
# the operator what happened if the snapshot upload fails.
TEMP_LOG="/tmp/litestream_deploy_$$.log"

# Run Litestream once and force a snapshot so this workflow completes only
# after the replacement database has been fully published to the replica path.
set +e
litestream replicate -config "$LOCAL_CONFIG" -once -force-snapshot > >(tee "$TEMP_LOG") 2>&1
REPLICATE_EXIT_CODE=$?
set -e

if [ "$REPLICATE_EXIT_CODE" -ne 0 ]; then
    echo ""
    echo "❌ Replication failed with exit code $REPLICATE_EXIT_CODE"
    echo "📋 Log contents:"
    cat "$TEMP_LOG" 2>/dev/null
    echo "🧹 Clearing staged restore secret because cutover did not complete..."
    clear_staged_force_restore_secret || true
    echo "❌ Production remains stopped so the existing mounted database is not overwritten by a bad restore."
    echo "💡 When you are ready to bring production back, re-enable auto-start with: flyctl machine update $MACHINE_ID -a $APP_NAME --autostart=true --skip-start -y"
    echo "💡 If you need to bring production back on the old database, make sure LITESTREAM_FORCE_RESTORE is not staged before starting the machine."
    exit 1
fi

echo ""
echo "✅ Replacement snapshot upload completed"
echo "📋 Next step: Re-enabling auto-start and starting production..."
start_production_app
exit 0
