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

## Database Backups

Litestream continuously replicates your SQLite database to S3-compatible storage.

### Monitoring Backups

```bash
# SSH into your app
flyctl ssh console

# Check litestream status
litestream snapshots -config /etc/litestream.yml /pb/pb_data/data.db
```

### Restore from Backup

**If database is corrupted:**

```bash
# SSH into your app
flyctl ssh console

# Stop the app temporarily
pkill tybalt

# Restore from latest backup
litestream restore -config /etc/litestream.yml -if-replica-exists /pb/pb_data/data.db

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
