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

**Purpose**: Deploys your local database to production via S3 replication.

**Usage**:

```bash
./scripts/deploy-local-db.sh
```

**What it does**:

1. Validates environment variables are set
2. Marks production to restore on next boot (`/app/pb_data/.force-restore`)
3. Stops the production machine
4. Replicates local database to S3
5. Starts the production machine (startup restores + resets Litestream/WAL state)

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

The production app requires a database in S3 to start. Push your local database before deploying:

```bash
source scripts/setup-env.sh
litestream replicate -config litestream.local.yml
# Wait 30-60 seconds for sync, then Ctrl+C
flyctl deploy
```

### Download production database

```bash
litestream restore -config litestream.local.yml -o ~/prod-backup.db app/pb_data/data.db
```

### Deploy local database to production

```bash
./scripts/deploy-local-db.sh
```

## Safety Notes

- **⚠️ Production requires S3 backup**: The app will fail to start if no database backup exists in S3
- **⚠️ Destructive operation**: `deploy-local-db.sh` replaces your entire production database
- **🔒 Environment required**: Must run `setup-env.sh` first

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
