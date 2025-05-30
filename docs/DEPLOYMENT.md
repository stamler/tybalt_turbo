# Tybalt Deployment Guide

This guide covers deploying Tybalt with Docker/Podman, including HTTPS support with Let's Encrypt.

## Quick Start

### HTTP Only (Development)

```bash
./scripts/docker-run.sh
```

Access at: <http://localhost:8080>

### HTTPS with Let's Encrypt (Production)

```bash
./scripts/docker-run.sh --domain yourdomain.com
```

Access at: <https://yourdomain.com>

## HTTPS Requirements

To use Let's Encrypt for automatic HTTPS certificates, you need:

1. **Domain name** pointing to your server's public IP
2. **Open ports** 80 and 443 from the internet
3. **No conflicting services** on ports 80/443
4. **Valid DNS** - domain must resolve to your server

## Deployment Options

### Option 1: Docker Compose (Recommended)

```bash
# HTTP only
docker-compose up -d

# HTTPS with domain
DOMAIN=example.com docker-compose up -d

# Using environment file
echo "DOMAIN=example.com" > .env
docker-compose up -d
```

**Benefits**: Industry standard, version controlled, reproducible, extensible

### Option 2: Simple Script (Convenience)

```bash
# HTTP only
./scripts/docker-run.sh --http-only

# HTTPS with your domain
./scripts/docker-run.sh --domain example.com

# View help
./scripts/docker-run.sh --help
```

**Benefits**: Auto-detects docker/podman, interactive warnings, beginner-friendly

### Option 3: Manual Docker/Podman

```bash
# Build
podman build -t tybalt .

# HTTP only
podman run -d --name tybalt -p 8080:8080 -v ./data:/pb/pb_data tybalt

# HTTPS
podman run -d --name tybalt \
  -p 80:80 -p 443:443 -p 8080:8080 \
  -v ./data:/pb/pb_data \
  -e DOMAIN=example.com \
  tybalt
```

**Benefits**: Full control, no additional tools required

## Let's Encrypt Process

When you specify a domain, PocketBase automatically:

1. **Binds to port 80** for HTTP challenge
2. **Binds to port 443** for HTTPS traffic  
3. **Requests certificate** from Let's Encrypt
4. **Redirects HTTP to HTTPS** automatically
5. **Auto-renews** certificates before expiry

## Data Persistence

Your database and uploaded files are stored in `./data/` on the host machine:

```fixed
./data/
├── data.db           # SQLite database
├── logs.db          # Application logs  
└── storage/         # Uploaded files
```

**Important**: Always backup this directory!

## Security Recommendations

Based on [PocketBase production guidelines](https://pocketbase.io/docs/going-to-production/):

### 1. Enable MFA for Superusers

- Go to Admin Dashboard > Settings > Auth
- Enable MFA for `_superusers` collection

### 2. Configure SMTP

- Set up external email service (SendGrid, Mailgun, etc.)
- Configure in Dashboard > Settings > Mail settings

### 3. Enable Rate Limiting

- Go to Dashboard > Settings > Application
- Configure rate limits for API endpoints

### 4. Settings Encryption (Optional)

```bash
# Generate 32-character key
export PB_ENCRYPTION_KEY="$(openssl rand -hex 16)"

# Add to docker-compose.yml environment
DOMAIN=example.com PB_ENCRYPTION_KEY="$PB_ENCRYPTION_KEY" docker-compose up -d
```

## Troubleshooting

### Let's Encrypt Issues

**Problem**: Certificate generation fails
**Solutions**:

- Verify domain points to server IP: `dig yourdomain.com`
- Check ports are open: `nmap -p 80,443 yourdomain.com`
- Ensure no other services use ports 80/443
- Check container logs: `podman logs tybalt`

**Problem**: "Too many requests" error
**Solution**: Let's Encrypt has rate limits. Wait before retrying.

### Container Issues

```bash
# View logs
podman logs tybalt

# Check container status
podman ps -a

# Restart container
podman restart tybalt

# Connect to container
podman exec -it tybalt sh
```

### Port Conflicts

If ports 80/443 are in use:

```bash
# Find what's using the ports
sudo lsof -i :80
sudo lsof -i :443

# Stop conflicting services
sudo systemctl stop apache2  # or nginx, etc.
```

## Production Checklist

- [ ] Domain points to server IP
- [ ] Ports 80 and 443 open in firewall
- [ ] SSL certificate obtained successfully
- [ ] Database backup strategy in place
- [ ] SMTP configured for email delivery
- [ ] Rate limiting enabled
- [ ] MFA enabled for superusers
- [ ] Monitoring/logging configured

## Backup & Recovery

### Manual Backup

```bash
# Stop container
podman stop tybalt

# Backup data
tar -czf backup-$(date +%Y%m%d).tar.gz data/

# Restart container  
podman start tybalt
```

### Automated Backup Script

```bash
#!/bin/bash
# backup-script.sh

BACKUP_NAME="backup-$(date +%Y%m%d-%H%M%S)"

# Stop application
docker-compose down

# Create backup
tar -czf "$BACKUP_NAME.tar.gz" data/

# Store backup safely
aws s3 cp "$BACKUP_NAME.tar.gz" s3://your-backup-bucket/
# or
rsync -av "$BACKUP_NAME.tar.gz" backup-server:/backups/

echo "Backup created: $BACKUP_NAME.tar.gz"

# Restart application
docker-compose up -d
```

### Built-in PocketBase Backups

For automated backups, PocketBase has built-in backup features accessible via the Admin Dashboard.

## Production Upgrades & Migrations

Tybalt uses PocketBase's migration system to handle database schema changes. When you develop new features that modify the database, you need to apply those migrations to production.

### How Migrations Work

1. **Auto-generation**: During development (`go run`), PocketBase automatically creates migration files when you change collections in the Admin UI
2. **Migration files**: Stored in `app/migrations/` with timestamps (e.g., `1748622851_add_imported_field_to_collections.go`)
3. **Forward & Rollback**: Each migration has both `up` and `down` functions for applying and reverting changes
4. **Sequential application**: Migrations are applied in timestamp order

### Production Upgrade Process

#### Option 1: Automated Deploy Script (Recommended)

```bash
# One command does it all - backup, build, migrate, deploy
./scripts/deploy.sh deploy
```

This script automatically:

- ✅ Creates timestamped backup
- ✅ Builds new image with latest code  
- ✅ Checks for pending migrations
- ✅ Applies migrations safely
- ✅ Verifies application health
- ✅ Auto-rollback on failure

**Other commands:**

```bash
./scripts/deploy.sh backup                           # Create backup only
./scripts/deploy.sh status                           # Check app status
./scripts/deploy.sh rollback ./backups/backup-xxx.tar.gz # Rollback
```

#### Option 2: Zero-Downtime with Docker Compose

```bash
# 1. Pull latest code with new migrations
git pull origin main

# 2. Build new image
docker-compose build

# 3. Stop and backup current data
docker-compose down
tar -czf backup-$(date +%Y%m%d-%H%M%S).tar.gz data/

# 4. Start with new image (migrations auto-apply)
docker-compose up -d

# 5. Verify application is healthy
docker-compose logs -f tybalt
```

#### Option 3: Manual Migration Control

```bash
# 1. Build new image
docker-compose build

# 2. Run migrations manually before starting
docker-compose run --rm tybalt ./tybalt migrate up

# 3. Start the application
docker-compose up -d
```

### Migration Commands

Your Tybalt application includes these migration commands:

```bash
# Apply all pending migrations
./tybalt migrate up

# Rollback last migration  
./tybalt migrate down

# See migration status
./tybalt migrate list

# Apply specific number of migrations
./tybalt migrate up --step=5

# Rollback specific number of migrations
./tybalt migrate down --step=2
```

### Development Workflow

#### Creating New Migrations

1. **Run in development mode** (auto-migration enabled):

   ```bash
   cd app && go run main.go serve
   ```

2. **Make schema changes** in Admin UI at `http://localhost:8080/_/`

3. **Migration files auto-generated** in `app/migrations/`

4. **Commit migrations** to version control:

   ```bash
   git add app/migrations/
   git commit -m "Add user profile fields migration"
   ```

#### Testing Migrations

```bash
# Test migration rollback
./tybalt migrate down --step=1

# Test migration forward  
./tybalt migrate up --step=1

# Run tests after migration
go test ./...
```

### Production Safety

#### Migration Validation

```bash
# Check pending migrations before deploy
docker-compose run --rm tybalt ./tybalt migrate list

# Test migrations on backup data
cp -r data/ data-test/
docker run --rm -v ./data-test:/pb/pb_data tybalt ./tybalt migrate up
```

#### Rollback Plan

If deployment fails:

```bash
# 1. Stop failed deployment
docker-compose down

# 2. Restore backup
rm -rf data/
tar -xzf backup-YYYYMMDD-HHMMSS.tar.gz

# 3. Start previous version
git checkout previous-tag
docker-compose up -d
```

### Common Migration Scenarios

#### Adding a New Field

Migration file automatically created:

```go
// Add field
collection.Fields.AddMarshaledJSONAt(-1, []byte(`{
    "id": "text123456789",
    "name": "new_field", 
    "type": "text",
    "required": false
}`))
```

#### Updating Business Rules

Migration file automatically created:

```go
// Update collection rules
json.Unmarshal([]byte(`{
    "createRule": "new_create_rule_here",
    "updateRule": "new_update_rule_here" 
}`), &collection)
```

#### Data Transformations

Custom migration:

```go
m.Register(func(app core.App) error {
    records, err := app.FindRecordsByExpr("users", nil)
    if err != nil {
        return err
    }
    
    for _, record := range records {
        // Transform data
        record.Set("new_field", transform(record.Get("old_field")))
        if err := app.Save(record); err != nil {
            return err
        }
    }
    return nil
}, func(app core.App) error {
    // Rollback logic
    return nil
})
```

### Monitoring Migrations

```bash
# Check migration status
docker-compose exec tybalt ./tybalt migrate list

# View recent logs
docker-compose logs --tail=100 tybalt

# Check database integrity
docker-compose exec tybalt sqlite3 /pb/pb_data/data.db "PRAGMA integrity_check;"
```

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Deploy to server
        run: |
          ssh production-server << 'EOF'
            cd /path/to/tybalt
            git pull origin main
            docker-compose down
            tar -czf backup-$(date +%Y%m%d-%H%M%S).tar.gz data/
            docker-compose up -d --build
          EOF
```
