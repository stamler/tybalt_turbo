#!/bin/sh
set -e

###############################################################################
# Startup & recovery behavior (Fly Machines + SQLite WAL + Litestream)
#
# This app stores its SQLite database at:
#   /app/pb_data/data.db
#
# You may run this either:
# - on the Machine's ephemeral filesystem (wiped on deploy/restart), or
# - on an attached Fly Volume (persists across deploy/restart) if you mount one.
#
# SQLite normally runs in WAL mode, which also creates:
#   /app/pb_data/data.db-wal
#   /app/pb_data/data.db-shm
#
# Litestream maintains local replication state on disk:
#   /app/pb_data/.data.db-litestream
#
# Why this matters:
# - Fly restarts/deploys recreate the root filesystem ("ephemeral storage wipe").
#   However, this does not protect you from WAL/state mismatches because:
#     - The DB can be restored/replaced/rewritten within the same Machine lifecycle.
#     - If you ever mount /app/pb_data on a volume, it persists across restarts.
#     - Partial restores or manual file operations can leave stale WAL/state behind.
# - If the database file is restored/replaced/rewritten (or WAL is truncated/reset)
#   but Litestream’s local state directory is left behind, Litestream can get stuck
#   in a loop with errors like:
#     "cannot verify wal state: prev WAL offset is less than the header size: -4088"
#   (this typically indicates the WAL stream Litestream expects no longer matches
#   the WAL currently on disk).
#
# What this script does differently than before:
# 1) It can perform a *clean restore* not only when data.db is missing, but also
#    when explicitly requested via:
#      - a flag file: /app/pb_data/.force-restore
#      - an env var:  LITESTREAM_FORCE_RESTORE=1
# 2) Before restoring, it deletes *all* local state that can cause WAL mismatch:
#      - /app/pb_data/.data.db-litestream
#      - /app/pb_data/data.db-wal
#      - /app/pb_data/data.db-shm
#      - /app/pb_data/data.db (so restore is authoritative)
# 3) It runs the application *under* Litestream using `litestream replicate -exec`.
#    That keeps process lifecycles coordinated (signals/shutdown), instead of
#    running Litestream in the background as a separate process.
#
# Operational note:
# - A forced restore is intentionally destructive to the local volume contents for
#   this DB (it replaces the on-volume DB with the replica contents).
# - If the replica is unavailable or credentials are wrong, the container will exit
#   (because `set -e`), which is safer than silently starting with a blank DB.
###############################################################################

log() {
  echo "[start] $*"
}

DB_PATH="/app/pb_data/data.db"
WAL_PATH="${DB_PATH}-wal"
SHM_PATH="${DB_PATH}-shm"
LITESTREAM_STATE_DIR="/app/pb_data/.data.db-litestream"
FORCE_RESTORE_FILE="/app/pb_data/.force-restore"

restore_needed=0
restore_reasons=""
if [ ! -f "$DB_PATH" ]; then
  restore_needed=1
  restore_reasons="${restore_reasons} missing-db"
fi
if [ -f "$FORCE_RESTORE_FILE" ]; then
  restore_needed=1
  restore_reasons="${restore_reasons} force-file"
fi
if [ "${LITESTREAM_FORCE_RESTORE:-}" = "1" ]; then
  restore_needed=1
  restore_reasons="${restore_reasons} env"
fi

if [ "$restore_needed" -eq 1 ]; then
  log "Restore requested:${restore_reasons}"
  log "Preparing clean restore (clearing local WAL + Litestream state if present)..."

  if [ -d "$LITESTREAM_STATE_DIR" ]; then
    log "Deleting ${LITESTREAM_STATE_DIR}"
    rm -rf "$LITESTREAM_STATE_DIR" || true
  else
    log "Not deleting ${LITESTREAM_STATE_DIR} (not present)"
  fi

  if [ -f "$WAL_PATH" ]; then
    log "Deleting ${WAL_PATH}"
    rm -f "$WAL_PATH" || true
  else
    log "Not deleting ${WAL_PATH} (not present)"
  fi

  if [ -f "$SHM_PATH" ]; then
    log "Deleting ${SHM_PATH}"
    rm -f "$SHM_PATH" || true
  else
    log "Not deleting ${SHM_PATH} (not present)"
  fi

  if [ -f "$DB_PATH" ]; then
    log "Deleting ${DB_PATH}"
    rm -f "$DB_PATH" || true
  else
    log "Not deleting ${DB_PATH} (not present)"
  fi

  log "Restoring ${DB_PATH} from Litestream replica (config: /etc/litestream.yml)"
  if litestream restore -config /etc/litestream.yml "$DB_PATH"; then
    log "Database restore succeeded"
  else
    log "ERROR: Database restore failed; refusing to start with a fresh/empty database."
    log "If you intentionally want a new empty DB, remove the restore requirement and start the app without Litestream."
    exit 1
  fi

  if command -v sqlite3 >/dev/null 2>&1; then
    log "Running integrity check on restored database (sqlite3 PRAGMA integrity_check)..."
    integrity_result=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;" 2>&1) || true
    if [ "$integrity_result" != "ok" ]; then
      log "ERROR: Integrity check failed after restore:"
      log "$integrity_result"
      log "Refusing to start with a corrupt database."
      exit 1
    fi
    log "Integrity check passed"
  else
    log "Skipping integrity check (sqlite3 CLI not installed in image)"
  fi

  if [ -f "$FORCE_RESTORE_FILE" ]; then
    log "Clearing restore flag ${FORCE_RESTORE_FILE}"
    rm -f "$FORCE_RESTORE_FILE" || true
  fi
else
  log "Using existing database at ${DB_PATH} (no restore requested)"
  log "Not deleting any database/WAL/Litestream state"
fi

# Run the app under Litestream so signals/shutdown are coordinated.
log "Starting Litestream + app (litestream replicate -exec)"
exec litestream replicate -config /etc/litestream.yml -exec "./tybalt serve --http=0.0.0.0:8080"
