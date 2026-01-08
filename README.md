# Tybalt 𝕋𝕌ℝ𝔹𝕆

A PocketBase-powered application with Svelte frontend, deployed on fly.io with litestream backups.

## Quick Start

### Development

**Important**: For authentication to work properly, you need to set the `PUBLIC_POCKETBASE_URL` environment variable in your UI development environment.

Create a `.env` file in the `ui/` directory:

```bash
# ui/.env
PUBLIC_POCKETBASE_URL=http://localhost:8090
```

Start the development servers:

```bash
# Backend
cd app && go run main.go serve

# Frontend (in separate terminal)
cd ui && npm run dev
```

### Production Deployment

```bash
flyctl deploy
```

## Project Structure

```text
/
├── app/                    # Go backend (PocketBase)
│   ├── migrations/         # Database migrations
│   ├── routes/             # API routes
│   ├── hooks/              # Database hooks
│   └── main.go             # Application entry point
├── ui/                     # Svelte frontend
│   ├── src/                # Svelte source code
│   └── build/              # Built UI assets
├── scripts/                # Database management scripts
│   ├── setup-env.sh        # Environment variable setup
│   ├── deploy-local-db.sh  # Deploy local database to production
│   └── README.md           # Scripts documentation
├── docs/                   # Documentation
│   └── DEPLOYMENT.md       # Deployment guide
├── Dockerfile              # Container build for fly.io
├── start.sh                # Container startup script
├── fly.toml                # Fly.io deployment configuration
├── litestream.yml          # Database replication configuration (production)
├── litestream.local.yml    # Database replication configuration (local)
└── .dockerignore           # Container build exclusions
```

## Setup

### Prerequisites

- Go 1.24+
- Node.js 20+
- npm
- [Fly.io CLI](https://fly.io/docs/hands-on/install-flyctl/)

### Installation

1. Clone and install dependencies:

```bash
git clone https://github.com/stamler/tybalt_turbo.git
cd tybalt_turbo

# Backend dependencies
cd app && go mod download

# Frontend dependencies
cd ../ui && npm install
```

2. Run locally:

```bash
# Terminal 1: Backend
cd app && go run main.go serve --dir="./test_pb_data"

# Terminal 2: Frontend
cd ui && npm run dev
```

App available at `http://localhost:5173`

## Deployment

### Initial Setup

1. **Create fly.io app:**

```bash
flyctl launch --no-deploy
```

2. **Set up S3 storage for backups:**

```bash
# Option 1: Use Tigris (Fly.io's S3-compatible storage)
flyctl storage create

# Option 2: Use AWS S3 (create bucket manually)
```

3. **Configure secrets:**

```bash
# For Tigris
flyctl secrets set \
  LITESTREAM_ACCESS_KEY_ID=your-tigris-key \
  LITESTREAM_SECRET_ACCESS_KEY=your-tigris-secret \
  LITESTREAM_BUCKET=your-bucket-name \
  LITESTREAM_REGION=us-east-1

# For AWS S3
flyctl secrets set \
  LITESTREAM_ACCESS_KEY_ID=your-aws-key \
  LITESTREAM_SECRET_ACCESS_KEY=your-aws-secret \
  LITESTREAM_BUCKET=your-s3-bucket \
  LITESTREAM_REGION=your-bucket-region \
  LITESTREAM_ENDPOINT=https://s3.amazonaws.com
```

4. **Push initial database to S3:**

The app requires a database backup in S3 to start. Push your local database:

```bash
source scripts/setup-env.sh
litestream replicate -config litestream.local.yml
# Wait 30-60 seconds, then Ctrl+C
```

5. **Deploy:**

```bash
flyctl deploy
```

See [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) for detailed deployment instructions.

### Updates

Just deploy:

```bash
flyctl deploy
```

Migrations are applied automatically on startup.

## Database Backups

Litestream continuously replicates your SQLite database to S3-compatible storage. No manual backups needed.

### Configuration

This project uses two litestream configuration files:

- **`litestream.yml`** - Production config (absolute paths for Docker)
- **`litestream.local.yml`** - Local development config (relative paths)

See [`scripts/README.md`](scripts/README.md) for complete litestream command documentation.

### Local Development

```bash
# Download production database locally
litestream restore -config litestream.local.yml -o ~/prod-backup.db app/pb_data/data.db
```

### Restore from backup

**Production:**

```bash
# Connect to your app
flyctl ssh console

# Restore from latest backup
litestream restore -if-replica-exists /app/pb_data/data.db
```

**Local:**

```bash
# Restore from backup to local database
litestream restore -config litestream.local.yml -if-replica-exists app/pb_data/data.db
```

## Testing

```bash
cd app && go test ./...
```

## License

MIT License - see [LICENSE](LICENSE) file.
