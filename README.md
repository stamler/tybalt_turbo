# Tybalt Turbo

A PocketBase-powered application with Svelte frontend, deployed on fly.io with litestream backups.

## Quick Start

### Development

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
├── docs/                   # Documentation
│   └── DEPLOYMENT.md       # Deployment guide
├── Dockerfile              # Container build for fly.io
├── start.sh                # Container startup script
├── fly.toml                # Fly.io deployment configuration
├── litestream.yml          # Database replication configuration
└── .dockerignore           # Container build exclusions
```

## Setup

### Prerequisites

- Go 1.23+
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

4. **Deploy:**

```bash
flyctl deploy
```

### Updates

Just deploy:

```bash
flyctl deploy
```

Migrations are applied automatically on startup.

## Database Backups

Litestream continuously replicates your SQLite database to S3-compatible storage. No manual backups needed.

**Restore from backup:**

```bash
# Connect to your app
flyctl ssh console

# Restore from latest backup
litestream restore -if-replica-exists /pb/pb_data/data.db
```

## Testing

```bash
cd app && go test ./...
```

## License

MIT License - see [LICENSE](LICENSE) file.
