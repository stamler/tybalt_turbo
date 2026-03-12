# Fly Litestream Runbook

Operational runbook for a Fly Machines deployment with:

- SQLite at `/app/pb_data/data.db`
- a persistent Fly volume mounted at `/app/pb_data`
- Litestream replicating to S3-compatible object storage

This app is intended to run as a single writable machine.

In the commands below, replace `<app-name>` with your Fly app name and `<machine-id>` with the current machine ID from `fly machine list`.

## Normal Restart

Use this when the app is healthy and you only need to restart the machine.

1. Find the machine ID:

```bash
fly machine list -a <app-name>
```

2. Restart the machine:

```bash
fly machine restart <machine-id> -a <app-name>
```

3. Watch logs:

```bash
fly logs -a <app-name>
```

Expected startup lines:

```text
[start] Using existing database at /app/pb_data/data.db (no restore requested)
[start] Not deleting any database/WAL/Litestream state
[start] Starting Litestream + app (litestream replicate -exec)
```

This confirms the machine reused the attached volume and did not restore from S3.

## Litestream Healthy Check

Check the app health endpoint:

```bash
fly ssh console -a <app-name> -C "wget -qO- http://127.0.0.1:8080/api/health"
```

Check Litestream metrics:

```bash
fly ssh console -a <app-name> -C "wget -qO- http://127.0.0.1:9091/metrics | head -100"
```

Healthy signs:

- `litestream_sync_error_count` stays at `0`
- `litestream_sync_count` increases over time when the app is writing
- `litestream_db_size` is non-zero
- replica `PUT` operations appear

## Litestream Unhealthy But Volume Intact

Use this when the machine is up but Litestream is failing or stuck.

### First response

1. Check logs:

```bash
fly logs -a <app-name>
```

2. Check metrics:

```bash
fly ssh console -a <app-name> -C "wget -qO- http://127.0.0.1:9091/metrics | rg 'litestream_sync|litestream_replica|litestream_compaction'"
```

3. Restart the machine:

```bash
fly machine restart <machine-id> -a <app-name>
```

Because the database is now on a Fly volume, a normal restart should preserve the local DB and avoid a full restore.

### If Litestream still fails after restart

With `auto-recover: true`, Litestream should attempt to reset its state after LTX/WAL mismatch errors. If it does not recover cleanly, force a fresh restore from the replica:

```bash
fly secrets set -a <app-name> --stage LITESTREAM_FORCE_RESTORE=1
fly machine restart <machine-id> -a <app-name>
fly secrets unset -a <app-name> --stage LITESTREAM_FORCE_RESTORE
```

Expected startup lines:

```text
[start] Restore requested: force-file
[start] Preparing clean restore (clearing local WAL + Litestream state if present)...
[start] Restoring /app/pb_data/data.db from Litestream replica (config: /etc/litestream.yml)
```

This deletes the local DB, WAL/SHM files, and Litestream state before restoring from S3.

## Volume Failure Or Lost Machine

Use this when the machine or attached volume is lost and the local DB cannot be reused.

1. Create and attach a replacement volume in the app's primary region.
2. Deploy the app.
3. On first boot, startup should restore from Litestream automatically because `/app/pb_data/data.db` is missing.
4. Watch logs for:

```text
[start] Restore requested: missing-db
[start] Restoring /app/pb_data/data.db from Litestream replica (config: /etc/litestream.yml)
```

5. After the app is up, verify:

```bash
fly ssh console -a <app-name> -C "df -h /app/pb_data"
fly ssh console -a <app-name> -C "wget -qO- http://127.0.0.1:8080/api/health"
fly ssh console -a <app-name> -C "wget -qO- http://127.0.0.1:9091/metrics | head -100"
```

## Force Restore From A Local Database Push

Use this when intentionally replacing production with a local database that you have already pushed to the Litestream replica.

1. Mark the machine to restore on next boot:

```bash
fly secrets set -a <app-name> --stage LITESTREAM_FORCE_RESTORE=1
```

2. Restart the machine:

```bash
fly machine restart <machine-id> -a <app-name>
fly secrets unset -a <app-name> --stage LITESTREAM_FORCE_RESTORE
```

The startup script will perform a clean restore from the replica when `LITESTREAM_FORCE_RESTORE=1` is present at boot.

This is the correct replacement workflow when the production machine already has a database on the mounted Fly volume. A normal restart will keep the on-volume database, so replacing production requires `LITESTREAM_FORCE_RESTORE=1`.

## Notes

- The Fly volume speeds up normal restarts and deploys by keeping the local SQLite DB on disk.
- Litestream remains the off-machine recovery path.
- Fly volumes can be extended later without deleting data, but not shrunk.
- This setup is still single-writer. Do not run multiple machines against the same SQLite DB.
