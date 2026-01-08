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

**⚠️ Important**: The production app will fail to start if no database backup exists in S3. This is intentional - it prevents the app from accidentally creating a blank database.

Before your first deployment, you must push a database to S3:

1. **Set up environment variables:**

   ```bash
   source scripts/setup-env.sh
   ```

2. **Push your local database to S3:**

   ```bash
   litestream replicate -config litestream.local.yml
   ```

   Let this run for 30-60 seconds to ensure the backup completes, then press Ctrl+C.

3. **Deploy the app:**

   ```bash
   flyctl deploy
   ```

The app will restore the database from S3 on first startup.

### Deploying Database Changes to Production

When you need to push local database changes (schema changes, data fixes, etc.) to production:

1. **Set up environment variables:**

   ```bash
   source scripts/setup-env.sh
   ```

2. **Stop the production app** (prevents conflicts):

   ```bash
   flyctl machine stop
   ```

3. **Push your local database to S3:**

   ```bash
   litestream replicate -config litestream.local.yml
   ```

   Let this run for 30-60 seconds, then press Ctrl+C.

4. **Delete the production database volume** (forces restore from S3):

   ```bash
   flyctl ssh console -C "rm /app/pb_data/data.db"
   ```

5. **Start the production app:**

   ```bash
   flyctl machine start
   ```

Alternatively, use the script:

```bash
./scripts/deploy-local-db.sh
```

#### Use Cases

- **Initial deployment** with seed data and superuser
- **Fixing corrupted production data**
- **Rolling back** to a known good state

#### ⚠️ Important Notes

- This replaces the **entire production database** with your local one
- Make sure your local database contains the state you want in production
- Always test locally before deploying database changes

## Database Backups

Litestream continuously replicates your SQLite database to S3-compatible storage.

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
# SSH into your app
flyctl ssh console

# Stop the app temporarily
pkill tybalt

# Restore from latest backup
litestream restore -config /etc/litestream.yml -if-replica-exists /app/pb_data/data.db

# Restart (or restart the machine from outside)
/start.sh &
```

**For complete disaster recovery:**

1. Create a new fly.io app
2. Configure the same secrets
3. Deploy
4. The database will automatically restore from the latest backup

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

**App not starting:**

- Check logs: `flyctl logs`
- Verify all required secrets are set: `flyctl secrets list`

**Performance issues:**

- Scale up: `flyctl scale vm performance-2x`
- Add more memory: `flyctl scale memory 2048`
