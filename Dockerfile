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

WORKDIR /app/ui

# Copy package files
COPY ui/package.json ui/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy UI source code
COPY ui/ ./

# Generate version info for UI (after copying source code)
RUN mkdir -p src/lib && \
    cd /app && ./scripts/version.sh ts > ui/src/lib/version.ts || \
    echo "// Fallback version\nexport const VERSION_INFO = {\n  name: 'Tybalt Turbo',\n  version: '0.1',\n  build: 1,\n  fullVersion: '0.1.1',\n  gitCommit: 'unknown',\n  gitCommitShort: 'unknown',\n  gitBranch: 'unknown',\n  buildTime: '$(date -u +"%Y-%m-%dT%H:%M:%SZ")'\n} as const;\n\nexport type VersionInfo = typeof VERSION_INFO;" > src/lib/version.ts

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

# Accept git info as build args (only needed for Go stage)
ARG GIT_COMMIT
ARG GIT_COMMIT_SHORT
ARG GIT_BRANCH

# Set them as environment variables for the version script
ENV GIT_COMMIT=${GIT_COMMIT}
ENV GIT_COMMIT_SHORT=${GIT_COMMIT_SHORT}
ENV GIT_BRANCH=${GIT_BRANCH}

# Copy version files and scripts
COPY version.json ./
COPY scripts/version.sh ./scripts/
RUN chmod +x scripts/version.sh

# Copy go mod files
COPY app/go.mod app/go.sum ./

# Download dependencies
RUN go mod download

# Copy Go source code
COPY app/ ./

# Generate version info for Go (after copying source code)
RUN mkdir -p constants && \
    ./scripts/version.sh go > constants/version.go || \
    (echo 'package constants'; \
     echo ''; \
     echo 'const ('; \
     echo '    AppName        = "Tybalt Turbo"'; \
     echo '    Version        = "0.1"'; \
     echo '    Build          = 1'; \
     echo '    FullVersion    = "0.1.1"'; \
     echo '    GitCommit      = "unknown"'; \
     echo '    GitCommitShort = "unknown"'; \
     echo '    GitBranch      = "unknown"'; \
     echo '    BuildTime      = "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"'; \
     echo ')') > constants/version.go

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