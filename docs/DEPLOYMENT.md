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

#### Option A: Tigris (Fly.io's S3-compatible storage)**

```bash
flyctl storage create
```

Note the credentials provided.

#### Option B: AWS S3**

Create an S3 bucket manually and note your access keys.

### 3. Configure Secrets

**For Tigris:**

```bash
flyctl secrets set \
  LITESTREAM_ACCESS_KEY_ID=your-tigris-access-key \
  LITESTREAM_SECRET_ACCESS_KEY=your-tigris-secret-key \
  LITESTREAM_BUCKET=your-bucket-name \
  LITESTREAM_REGION=us-east-1
```

**For AWS S3:**

```bash
flyctl secrets set \
  LITESTREAM_ACCESS_KEY_ID=your-aws-access-key \
  LITESTREAM_SECRET_ACCESS_KEY=your-aws-secret-key \
  LITESTREAM_BUCKET=your-s3-bucket-name \
  LITESTREAM_REGION=your-bucket-region \
  LITESTREAM_ENDPOINT=https://s3.amazonaws.com
```

### 4. Deploy

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
# Check database generations/snapshots
litestream generations -config litestream.local.yml

# Check what's backed up in S3
litestream snapshots -config litestream.local.yml

# Replicate local database to S3
litestream replicate -config litestream.local.yml
```

**Alternative**: You can also query S3 directly (works regardless of config):

```bash
# Check generations directly from S3
litestream generations -replica s3://${LITESTREAM_BUCKET}/tybalt

# Check snapshots directly from S3  
litestream snapshots -replica s3://${LITESTREAM_BUCKET}/tybalt
```

### Deploying Local Database Changes to Production

When you need to push local database changes (schema changes, seed data, etc.) to production:

#### Setup for Database Deployment

1. **Install Litestream locally:**

   ```bash
   brew install benbjohnson/litestream/litestream
   ```

2. **Get environment variables from your Fly app:**

   ```bash
   flyctl machine exec --app your-app-name MACHINE_ID -- printenv | grep LITESTREAM
   ```

3. **Set environment variables locally:**

   ```bash
   export LITESTREAM_ACCESS_KEY_ID="your_access_key"
   export LITESTREAM_SECRET_ACCESS_KEY="your_secret_key"  
   export LITESTREAM_BUCKET="your_bucket_name"
   export LITESTREAM_ENDPOINT="your_endpoint"
   export LITESTREAM_REGION="your_region"
   ```

#### Push Local Database to Production

1. **Push your local database to S3:**

   ```bash
   litestream replicate -config litestream.local.yml
   ```

   Let this run for 30-60 seconds to ensure the backup completes.

2. **Restart the Fly app to pick up the new database:**

   ```bash
   flyctl apps restart your-app-name
   ```

#### Use Cases

- **Fixing corrupted production data**
- **Rolling back** to a known good state
- **Initial seeding** (including superuser setup)

#### ⚠️ Important Notes

- This replaces the **entire production database** with your local one
- Make sure your local database contains the state you want in production
- Old backups are retained according to retention policy (72 hours by default)
- Always test locally before deploying database changes

## Database Backups

Litestream continuously replicates your SQLite database to S3-compatible storage.

### Monitoring Backups

```bash
# SSH into your app
flyctl ssh console

# Check litestream status
litestream snapshots -config /etc/litestream.yml /app/pb_data/data.db
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
