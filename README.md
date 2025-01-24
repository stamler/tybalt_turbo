# Tybalt Turbo

A modern, fast implementation of Tybalt built with Go and SvelteKit.

## Project Structure

- `app/` - Backend application written in Go using the PocketBase framework
- `ui/` - Frontend application written in SvelteKit with Svelte 5
- `descriptions/` - Project documentation and descriptions
- `import_data/` - Data import utilities
- `prompts/` - System prompts and configurations

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

## Development

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
