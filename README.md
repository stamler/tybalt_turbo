# Tybalt Turbo

A PocketBase-powered application with Svelte frontend.

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
├── scripts/                # Deployment and utility scripts
│   ├── deploy.sh           # Production deployment script
│   └── docker-run.sh       # Docker/Podman convenience script
├── docs/                   # Documentation
│   └── DEPLOYMENT.md       # Deployment guide
├── Dockerfile              # Container build instructions
├── docker-compose.yml      # Container orchestration
└── .dockerignore           # Docker build exclusions
```

## Key Features

- **PocketBase backend** with custom Go extensions
- **Svelte frontend** with modern UI components
- **Docker/Podman support** for containerized deployment
- **Automatic migrations** for database schema changes
- **HTTPS support** with Let's Encrypt integration
- **Production-ready** deployment scripts with backup/rollback

## Quick Start

### Production Deployment

See [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) for comprehensive deployment instructions.

**Quick deployment:**

```bash
./scripts/deploy.sh deploy
```

### Development Setup

**Quick start:**

```bash
# Backend
cd app && go run main.go serve

# Frontend  
cd ui && npm run dev
```

## Getting Started

### Prerequisites

- Go 1.23 or later
- Node.js 20 or later
- npm or pnpm

### Installation

1. Clone the repository:

```bash
git clone https://github.com/stamler/tybalt_turbo.git
cd tybalt_turbo
```

2. Set up the backend:

```bash
cd app
go mod download
```

3. Set up the frontend:

```bash
cd ui
npm install  # or pnpm install
```

### Running the Application

1. Start the backend server with the test database:

```bash
cd app
go run main.go serve --dir="./test_pb_data"
```

2. In a separate terminal, start the frontend development server:

```bash
cd ui
npm run dev
```

The application will be available at `http://localhost:5173` by default.

### Testing

To run the Go backend tests:

```bash
cd app
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
