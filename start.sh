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
# - Deploys/restarts can stop processes abruptly.
# - Even without a persistent volume, the database can be restored/replaced/rewritten
#   within the same Machine lifecycle (or after a partial restore attempt).
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

DB_PATH="/app/pb_data/data.db"
WAL_PATH="${DB_PATH}-wal"
SHM_PATH="${DB_PATH}-shm"
LITESTREAM_STATE_DIR="/app/pb_data/.data.db-litestream"
FORCE_RESTORE_FILE="/app/pb_data/.force-restore"

restore_needed=0
if [ ! -f "$DB_PATH" ]; then
  restore_needed=1
fi
if [ -f "$FORCE_RESTORE_FILE" ]; then
  restore_needed=1
fi
if [ "${LITESTREAM_FORCE_RESTORE:-}" = "1" ]; then
  restore_needed=1
fi

if [ "$restore_needed" -eq 1 ]; then
  echo "Restoring database from Litestream..."

  # If the database was deleted/replaced without clearing Litestream/WAL state then
  # Litestream can get stuck with WAL verification errors on boot.
  rm -rf "$LITESTREAM_STATE_DIR" || true
  rm -f "$WAL_PATH" "$SHM_PATH" || true
  rm -f "$DB_PATH" || true

  litestream restore -config /etc/litestream.yml "$DB_PATH"
  rm -f "$FORCE_RESTORE_FILE" || true

  echo "Database restored successfully"
fi

# Run the app under Litestream so signals/shutdown are coordinated.
exec litestream replicate -config /etc/litestream.yml -exec "./tybalt serve --http=0.0.0.0:8080"
