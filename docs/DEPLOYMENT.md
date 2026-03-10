# Tybalt Deployment Guide

Deploy Tybalt to fly.io with continuous database backups via litestream.

## Prerequisites

- [Fly.io CLI](https://fly.io/docs/hands-on/install-flyctl/) installed
- Fly.io account with billing enabled

## Initial Setup

### 1. Create Fly App

```bash
flyctl launch --no-deploy
```

This creates `fly.toml` - update the app name and region as needed.

### 2. Set Up Storage for Backups

#### Option A: Tigris (Fly.io's S3-compatible storage)

```bash
flyctl storage create
```

Note the credentials provided, then configure secrets:

```bash
flyctl secrets set \
  LITESTREAM_ACCESS_KEY_ID=your-tigris-access-key \
  LITESTREAM_SECRET_ACCESS_KEY=your-tigris-secret-key \
  LITESTREAM_BUCKET=your-bucket-name \
  LITESTREAM_REGION=us-east-1
```

#### Option B: AWS S3

Create an S3 bucket manually and configure secrets:

```bash
flyctl secrets set \
  LITESTREAM_ACCESS_KEY_ID=your-aws-access-key \
  LITESTREAM_SECRET_ACCESS_KEY=your-aws-secret-key \
  LITESTREAM_BUCKET=your-s3-bucket-name \
  LITESTREAM_REGION=your-bucket-region \
  LITESTREAM_ENDPOINT=https://s3.amazonaws.com
```

### 3. Deploy

```bash
flyctl deploy
```

Your app will be available at `https://your-app-name.fly.dev`

## Updates

Deploy changes:

```bash
flyctl deploy
```

Database migrations are applied automatically on startup.

## Database Management

### Configuration Files

This project uses two litestream configuration files:

- **`litestream.yml`** - Production config with absolute paths (used in Docker)
- **`litestream.local.yml`** - Local development config with relative paths

### Local Development Commands

For local development, use the local config file:

```bash
# Download the latest production database
litestream restore -config litestream.local.yml -o ~/prod-backup.db app/pb_data/data.db

# Restore to a specific point in time
litestream restore -config litestream.local.yml -timestamp 2025-01-08T12:00:00Z -o ~/prod-backup.db app/pb_data/data.db

# Replicate local database to S3
litestream replicate -config litestream.local.yml
```

### Initial Database Deployment

**⚠️ Important**: The production app will fail to start if startup needs to restore and no Litestream replica exists in S3. This is intentional and prevents the app from accidentally creating a blank database.

This is the workflow for an initial deployment where production has no local database yet. On first boot, startup sees that `/app/pb_data/data.db` is missing and restores from the Litestream replica into the mounted volume.

Before the first deploy, you must push a database to S3:

1. **Set up environment variables:**

   ```bash
   source scripts/setup-env.sh
   ```

2. **Clear any stale local Litestream state** (required before pushing from your local machine):

   ```bash
   rm -rf app/pb_data/.data.db-litestream
   ```

   This removes local replication metadata that may be out of sync with S3.

3. **Push your local database to the Litestream replica:**

   ```bash
   litestream replicate -config litestream.local.yml
   ```

   Let this run for 30-60 seconds to ensure the backup completes, then press Ctrl+C.

4. **Deploy the app:**

   ```bash
   flyctl deploy
   ```

The app will restore the database from S3 on first startup because no local production database exists yet.

### Deploying Database Changes to Production

This is the workflow for replacing an already-existing production database on the mounted Fly volume. Because normal restarts now reuse the on-volume database, replacing production requires an explicit forced restore on the next boot.

When you need to push local database changes (schema changes, data fixes, rollback state, etc.) to production:

1. **Set up environment variables:**

   ```bash
   source scripts/setup-env.sh
   ```

2. **Mark production to discard the on-volume DB and restore from S3 on next boot**:

   ```bash
   flyctl ssh console -C "touch /app/pb_data/.force-restore"
   ```

3. **Stop the production machine** (prevents production writes while you replace the replica contents):

   ```bash
   MACHINE_ID=$(flyctl status --json | jq -r '.Machines[0].id')
   flyctl machine stop $MACHINE_ID
   ```

4. **Push your local database to the Litestream replica:**

   ```bash
   litestream replicate -config litestream.local.yml
   ```

   Let this run for 30-60 seconds, then press Ctrl+C.

5. **Start the production machine**:

   ```bash
   flyctl machine start $MACHINE_ID
   ```

On next boot, the app startup script will:

- Ignore the existing on-volume database because `.force-restore` is present
- Delete `data.db`, WAL/SHM files, and local Litestream state from the volume
- Restore `data.db` from S3 into the mounted volume
- Clear stale Litestream state (`/app/pb_data/.data.db-litestream`) and any SQLite WAL/SHM files

This prevents common Litestream WAL verification errors after DB replacement.

Alternatively, use the script:

```bash
./scripts/deploy-local-db.sh
```

#### Use Cases

- **Replacing** an existing production database with a local one
- **Fixing corrupted production data**
- **Rolling back** to a known good state

#### ⚠️ Important Notes

- This replaces the **entire production database** with your local one
- Make sure your local database contains the state you want in production
- Always test locally before deploying database changes
- PocketBase backup restore should be treated as an offline source-generation step, not as an in-place restore on the live production machine

## Database Backups

Litestream continuously replicates your SQLite database to S3-compatible storage.

See [`docs/fly-litestream-runbook.md`](/Users/dean/code/tybalt_turbo/docs/fly-litestream-runbook.md) for the restart, Litestream recovery, force-restore, and volume-failure procedures.

### Monitoring Backups

```bash
# SSH into your app
flyctl ssh console

# List available backup files
litestream ltx -config /etc/litestream.yml /app/pb_data/data.db
```

### Restore from Backup

**If database is corrupted:**

```bash
# Mark the machine to discard the on-volume DB and restore from S3 on next boot
flyctl ssh console -C "touch /app/pb_data/.force-restore"

# Restart the machine so startup performs a clean restore
MACHINE_ID=$(flyctl status --json | jq -r '.Machines[0].id')
flyctl machine restart $MACHINE_ID
```

**For complete disaster recovery:**

1. Create a new fly.io app
2. Configure the same secrets
3. Deploy
4. The database will automatically restore from the latest backup on first boot because the local DB is missing

## Useful Commands

```bash
# View logs
flyctl logs

# Monitor app
flyctl status

# Scale up/down
flyctl scale count 2

# SSH into app
flyctl ssh console

# View secrets
flyctl secrets list
```

## Troubleshooting

**Database not restoring:**

- Check litestream logs: `flyctl logs -a your-app`
- Verify S3 credentials are correct
- Ensure bucket exists and is accessible

**Litestream WAL verification errors (e.g. `cannot verify wal state`)**

- Mark a clean restore on next boot: `flyctl ssh console -C "touch /app/pb_data/.force-restore"`
- Restart the machine: `flyctl machine restart $(flyctl status --json | jq -r '.Machines[0].id')`

**App not starting:**

- Check logs: `flyctl logs`
- Verify all required secrets are set: `flyctl secrets list`

**Performance issues:**

- Scale up: `flyctl scale vm performance-2x`
- Add more memory: `flyctl scale memory 2048`
