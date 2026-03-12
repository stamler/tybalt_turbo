# Database Scripts

This directory contains scripts for database operations for the Tybalt application.

## Scripts

### `setup-env.sh`

**Purpose**: Sets up environment variables needed for litestream operations by fetching them from your Fly.io app.

**Usage**:

```bash
source scripts/setup-env.sh
```

**What it does**:

- Fetches litestream secrets from your Fly.io app
- Sets environment variables locally (`LITESTREAM_*`)
- Validates the setup worked correctly

**Prerequisites**:

- `flyctl` CLI installed and authenticated
- Your Fly.io app must be running
- `jq` installed for JSON parsing

---

### `deploy-local-db.sh`

**Purpose**: Replaces the existing production database by pushing your local database to the Litestream replica and forcing production to restore it on next boot.

**Usage**:

```bash
./scripts/deploy-local-db.sh
```

**What it does**:

1. Validates environment variables are set
2. Stages `LITESTREAM_FORCE_RESTORE=1` so production restores on next boot
3. Disables Fly auto-start on the current production machine and stops that machine
4. Clears local Litestream state and forces a full replacement snapshot upload to S3
5. Re-enables Fly auto-start and starts the same production machine
6. Startup clears local DB/WAL/Litestream state from the volume and restores from the replica

---

## Litestream Commands

After running `source scripts/setup-env.sh`, you can use these litestream commands directly:

### List available backups

```bash
litestream ltx -config litestream.local.yml app/pb_data/data.db
```

### Download the latest production database

```bash
# Restore to a new file (preserves your local database)
litestream restore -config litestream.local.yml -o ~/prod-backup.db app/pb_data/data.db

# Or restore directly to local database (overwrites it)
litestream restore -config litestream.local.yml app/pb_data/data.db
```

### Restore to a specific point in time

```bash
litestream restore -config litestream.local.yml -timestamp 2025-01-08T12:00:00Z -o ~/prod-backup.db app/pb_data/data.db
```

## Workflow

### First-time setup

```bash
source scripts/setup-env.sh
```

### Initial deployment (first time only)

Use this when production does not yet have a local database on its Fly volume. Push your local database to the Litestream replica before deploying:

```bash
source scripts/setup-env.sh
rm -rf app/pb_data/.data.db-litestream
litestream replicate -config litestream.local.yml -once -force-snapshot
flyctl deploy
```

On first boot, startup restores from the Litestream replica because `/app/pb_data/data.db` is missing.

### Download production database

```bash
litestream restore -config litestream.local.yml -o ~/prod-backup.db app/pb_data/data.db
```

### Deploy local database to production

```bash
./scripts/deploy-local-db.sh
```

Use this when production already has a database on the mounted volume and you want to replace it. A plain restart will keep the existing on-volume DB, so replacement requires the forced-restore workflow.

## Safety Notes

- **⚠️ Startup restore requires S3 backup**: The app will fail to start if it needs to restore and no database backup exists in S3
- **⚠️ Destructive operation**: `deploy-local-db.sh` replaces your entire production database
- **🔒 Environment required**: Must run `setup-env.sh` first
- **⚠️ Do not restore over the live production DB in place**: replace production by updating the replica and staging `LITESTREAM_FORCE_RESTORE=1` on next boot

## Troubleshooting

### "bucket required for s3 replica"

```bash
# Solution: Set up environment variables
source scripts/setup-env.sh
```

### "No machines found"

```bash
# Solution: Make sure your Fly.io app is running
flyctl machine start
```

### "Failed to fetch secrets"

```bash
# Solution: Ensure you're authenticated with flyctl
flyctl auth login
```
