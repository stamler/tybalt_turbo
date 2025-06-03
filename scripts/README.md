# Database Rollback Scripts

This directory contains scripts for robust database rollback operations for the Tybalt application.

## Overview

These scripts provide a complete system for rolling back your production database to any previous generation stored in litestream backups. This is essential for deployment robustness and disaster recovery.

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

### `list-generations.sh`

**Purpose**: Lists all available database generations with timestamps.

**Usage**:

```bash
./scripts/list-generations.sh
```

**What it does**:

- Shows all available database backups/generations
- Displays generation IDs, timestamps, and lag times
- Provides helpful usage tips

**Output example**:

```
🗂️  Available database generations:

name  generation        lag       start                 end
s3    f1f5e9fd95acf3b6  -395ms    2025-06-02T20:46:17Z  2025-06-02T20:46:17Z
s3    aa46314586c7e98e  -22m17s   2025-06-02T21:07:33Z  2025-06-02T21:08:34Z

💡 To rollback: ./scripts/rollback.sh <generation_id>
```

---

### `rollback.sh`

**Purpose**: Complete automated rollback to a specific database generation.

**Usage**:

```bash
./scripts/rollback.sh <generation_id>
```

**Example**:

```bash
./scripts/rollback.sh f1f5e9fd95acf3b6
```

**What it does**:

1. Validates environment variables are set
2. Backs up current local database (safety measure)
3. Restores the specified generation locally
4. Pushes the restored database to S3
5. Restarts the production Fly.io app
6. Provides confirmation and app URL

**Safety features**:

- Environment variable validation
- Local database backup before rollback
- Error handling with `set -e`
- Timeout protection for replication
- Clear status messages throughout

## Workflow

### First-time setup

```bash
# 1. Set up environment variables
source scripts/setup-env.sh
```

### Regular rollback operations

```bash
# 2. List available generations
./scripts/list-generations.sh

# 3. Rollback to specific generation
./scripts/rollback.sh <generation_id>
```

## Use Cases

- **Deployment rollback**: Quickly revert after a bad deployment
- **Data corruption recovery**: Restore to a known good state
- **Superuser recovery**: Restore generation with admin credentials
- **Testing**: Roll back to specific test data states

## Safety Notes

- **⚠️ Destructive operation**: Rollback replaces your entire production database
- **📝 Always verify**: Check the generation timestamp before rolling back
- **💾 Automatic backup**: Local database is backed up before rollback
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

### Permission denied

```bash
# Solution: Make scripts executable
chmod +x scripts/*.sh
```

## Architecture

The rollback system works by:

1. **Local restore**: Downloads the specific generation to local database
2. **S3 replication**: Pushes the restored database back to S3 storage
3. **Production restart**: Fly.io app restarts and picks up the "new" backup
4. **Automatic sync**: Litestream resumes continuous replication

This approach ensures:

- ✅ Atomic operations (all-or-nothing)
- ✅ Production consistency
- ✅ Minimal downtime
- ✅ Audit trail preservation
