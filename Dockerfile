# Multi-stage Dockerfile for Tybalt PocketBase Application with Litestream

# Stage 1: Build the Svelte UI
FROM node:20-alpine AS ui-builder

WORKDIR /app/ui

# Copy package files
COPY ui/package.json ui/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy UI source code
COPY ui/ ./

# Build the UI
RUN npm run build

# Stage 2: Build the Go application
FROM golang:1.23-alpine AS go-builder

# Install git and other build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

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
WORKDIR /pb

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
RUN mkdir -p /pb/pb_data

# Expose port 8080
EXPOSE 8080

# Use the startup script
CMD ["/start.sh"] 