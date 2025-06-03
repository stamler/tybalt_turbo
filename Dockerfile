# Multi-stage Dockerfile for Tybalt PocketBase Application with Litestream

# Stage 1: Build the Svelte UI
FROM node:20-alpine AS ui-builder

WORKDIR /app

# Install jq for version processing
RUN apk add --no-cache jq git

# Copy version files and scripts
COPY version.json ./
COPY scripts/version.sh ./scripts/
RUN chmod +x scripts/version.sh

# Generate version info for UI
RUN ./scripts/version.sh ts > ui/src/lib/version.ts || echo "// Fallback version\nexport const VERSION_INFO = { version: '0.1.0', build: 1, name: 'Tybalt Turbo', fullVersion: '0.1.0.1', gitCommit: 'unknown', gitCommitShort: 'unknown', gitBranch: 'unknown', buildTime: '$(date -u +"%Y-%m-%dT%H:%M:%SZ")' } as const;" > ui/src/lib/version.ts

WORKDIR /app/ui

# Copy package files
COPY ui/package.json ui/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy UI source code
COPY ui/ ./

# Set the PocketBase URL for the production build
# This will be baked into the static build since SvelteKit uses adapter-static
ARG PUBLIC_POCKETBASE_URL
ENV PUBLIC_POCKETBASE_URL=${PUBLIC_POCKETBASE_URL}

# Build the UI
RUN npm run build

# Stage 2: Build the Go application
FROM golang:1.23-alpine AS go-builder

# Install git, jq and other build dependencies
RUN apk add --no-cache git ca-certificates jq

WORKDIR /app

# Copy version files and scripts
COPY version.json ./
COPY scripts/version.sh ./scripts/
RUN chmod +x scripts/version.sh

# Generate version info for Go
RUN ./scripts/version.sh go > app/constants/version.go || echo "package constants\nconst (\n\tAppName = \"Tybalt Turbo\"\n\tVersion = \"0.1.0\"\n\tBuild = 1\n\tFullVersion = \"0.1.0.1\"\n\tGitCommit = \"unknown\"\n\tGitCommitShort = \"unknown\"\n\tGitBranch = \"unknown\"\n\tBuildTime = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"\n)" > app/constants/version.go

# Copy go mod files
COPY app/go.mod app/go.sum ./

# Download dependencies
RUN go mod download

# Copy Go source code
COPY app/ ./

# Copy built UI from previous stage
COPY --from=ui-builder /app/ui/build ./pb_public

# Build the Go application
RUN CGO_ENABLED=0 go build -o tybalt main.go

# Stage 3: Final runtime image
FROM alpine:latest

# Install dependencies
RUN apk add --no-cache ca-certificates tzdata

# Install litestream
RUN wget https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.tar.gz -O - | tar -xzf - -C /usr/local/bin

# Create app directory
WORKDIR /app

# Copy the built application and UI
COPY --from=go-builder /app/tybalt ./tybalt

# Copy the UI assets
COPY --from=go-builder /app/pb_public ./pb_public

# Copy litestream configuration
COPY litestream.yml /etc/litestream.yml

# Copy startup script
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Create pb_data directory for database persistence
RUN mkdir -p /app/pb_data

# Expose port 8080
EXPOSE 8080

# Use the startup script
CMD ["/start.sh"] 